package overlay

import (
	context "context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	fmt "fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	proto "github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	any "github.com/golang/protobuf/ptypes/any"
	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

//go:generate protoc -I ./ --go_out=plugins=grpc:./ ./overlay.proto

// Identity is the private identity of the nodes.
type Identity struct {
	Port        string
	Certificate *tls.Certificate
}

// Peer is the public identity of the nodes.
type Peer struct {
	Port        string
	Certificate *x509.Certificate
}

// Roster is the set of nodes known.
type Roster []Peer

// Aggregation is the interface to implement to register a protocol that
// will contact all the nodes and aggregate their replies.
type Aggregation interface {
	Announce() proto.Message

	Spread(proto.Message) proto.Message

	Process([]proto.Message) proto.Message
}

// Collection is the interface to implement to register a protocol that
// will contact all the nodes and gather data for all of them.
type Collection interface {
	Prepare() proto.Message
}

// Overlay is the network abstraction to communicate with the roster.
type Overlay struct {
	*grpc.Server
	WebProxy *http.Server

	identity Identity
	roster   Roster

	aggregators map[string]Aggregation
	collectors  map[string]Collection
}

// NewOverlay returns a new overlay.
func NewOverlay(ident Identity) *Overlay {
	creds := credentials.NewServerTLSFromCert(ident.Certificate)
	srv := grpc.NewServer(grpc.Creds(creds))
	wrappedGrpc := grpcweb.WrapServer(srv, grpcweb.WithOriginFunc(func(origin string) bool {
		fmt.Println("origin", origin)
		return true
	}))
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{
			*ident.Certificate,
		},
	}

	port, err := strconv.Atoi(ident.Port[1:])
	if err != nil {
		log.Fatalf("error parsing port: %v\n", err)
	}
	port += 1000
	httpsServer := &http.Server{
		Addr: fmt.Sprintf(":%d", port),
		Handler: http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			if wrappedGrpc.IsGrpcWebRequest(req) {
				wrappedGrpc.ServeHTTP(res, req)
				return
			}
			if req.Method == http.MethodOptions && req.Header.Get("Access-Control-Request-Method") != "" {
				headers := res.Header()
				headers.Set("Access-Control-Allow-Origin", "*")
				headers.Set("Access-Control-Allow-Headers", "*")
				res.WriteHeader(http.StatusOK)
			} else {
				http.DefaultServeMux.ServeHTTP(res, req)
			}
		}),
		TLSConfig: tlsConfig,
	}

	overlay := &Overlay{
		Server:   srv,
		WebProxy: httpsServer,
		identity: ident,

		aggregators: make(map[string]Aggregation),
		collectors:  make(map[string]Collection),
	}

	RegisterOverlayServer(srv, &overlayService{Overlay: overlay})

	return overlay
}

func (o *Overlay) GetPeer() Peer {
	return Peer{
		Port:        o.identity.Port,
		Certificate: o.identity.Certificate.Leaf,
	}
}

func (o *Overlay) SetRoster(ro Roster) {
	o.roster = ro
}

func (o *Overlay) RegisterAggregation(name string, impl Aggregation) {
	o.aggregators[name] = impl
}

func (o *Overlay) RegisterCollection(name string, impl Collection) {
	o.collectors[name] = impl
}

func (o *Overlay) getPosition() int {
	for i, ident := range o.roster {
		if ident.Port == o.identity.Port {
			return i
		}
	}

	return -1
}

func (o *Overlay) getChildren(tree *Tree) []*grpc.ClientConn {
	children := tree.ChildrenOf(o.getPosition())
	clients := make([]*grpc.ClientConn, len(children))
	for i, idx := range children {
		pool := x509.NewCertPool()
		pool.AddCert(o.roster[idx].Certificate)

		creds := credentials.NewClientTLSFromCert(pool, "")

		address := fmt.Sprintf("localhost%s", o.roster[idx].Port)

		// Connection using TLS.
		conn, err := grpc.Dial(address, grpc.WithTransportCredentials(creds))
		if err != nil {
			log.Fatalf("did not connect: %v", err)
		}

		clients[i] = conn
	}

	return clients
}

func (o *Overlay) Aggregate(name string) (proto.Message, error) {
	agg := o.aggregators[name]
	if agg == nil {
		return nil, errors.New("aggregation not found")
	}

	ann, err := ptypes.MarshalAny(agg.Announce())
	if err != nil {
		return nil, err
	}

	msg := &AggregateAnnouncement{
		Id:      name,
		Tree:    &Tree{N: 5, K: 3, Root: int64(o.getPosition())},
		Message: ann,
	}
	replies, err := o.sendAggregate(msg)
	if err != nil {
		return nil, err
	}

	final := agg.Process(replies)

	return final, nil
}

func (o *Overlay) sendAggregate(msg *AggregateAnnouncement) ([]proto.Message, error) {
	replies := make([]proto.Message, 0)
	for _, conn := range o.getChildren(msg.Tree) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		client := NewOverlayClient(conn)
		defer conn.Close()

		reply, err := client.SendAggregateAnnouncement(ctx, msg)
		if err != nil {
			return nil, err
		}

		var da ptypes.DynamicAny
		err = ptypes.UnmarshalAny(reply.GetMessage(), &da)
		if err != nil {
			return nil, err
		}

		replies = append(replies, da.Message)
	}
	return replies, nil
}

func (o *Overlay) Collect(name string) ([]proto.Message, error) {
	msg := &CollectionRequest{
		Id:   name,
		Tree: &Tree{N: 5, K: 2, Root: int64(o.getPosition())},
	}

	replies, err := o.sendCollect(name, msg)
	if err != nil {
		return nil, err
	}

	messages := make([]proto.Message, len(replies))
	var da ptypes.DynamicAny
	for i, m := range replies {
		err = ptypes.UnmarshalAny(m, &da)
		if err != nil {
			return nil, err
		}

		messages[i] = da.Message
	}

	return messages, nil
}

func (o *Overlay) sendCollect(name string, msg *CollectionRequest) ([]*any.Any, error) {
	collector := o.collectors[name]
	if collector == nil {
		return nil, errors.New("collector not found")
	}

	replies := make([]*any.Any, 0)
	for _, conn := range o.getChildren(msg.Tree) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		client := NewOverlayClient(conn)
		defer conn.Close()

		reply, err := client.SendCollectionRequest(ctx, msg)
		if err != nil {
			return nil, err
		}

		replies = append(replies, reply.GetMessage()...)
	}

	own, err := ptypes.MarshalAny(collector.Prepare())
	if err != nil {
		return nil, err
	}

	replies = append(replies, own)

	return replies, nil
}

// gRPC service for the overlay.
type overlayService struct {
	UnimplementedOverlayServer

	Overlay *Overlay
}

func (o *overlayService) SendAggregateAnnouncement(ctx context.Context, in *AggregateAnnouncement) (*AggregateResponse, error) {
	agg := o.Overlay.aggregators[in.GetId()]
	if agg == nil {
		return nil, errors.New("aggregation not found")
	}

	var da ptypes.DynamicAny
	err := ptypes.UnmarshalAny(in.GetMessage(), &da)
	if err != nil {
		return nil, err
	}

	nextAnn, err := ptypes.MarshalAny(agg.Spread(da.Message))
	if err != nil {
		return nil, err
	}

	msg := &AggregateAnnouncement{
		Id:      in.GetId(),
		Tree:    in.GetTree(),
		Message: nextAnn,
	}

	replies, err := o.Overlay.sendAggregate(msg)
	if err != nil {
		return nil, err
	}

	r, err := ptypes.MarshalAny(agg.Process(replies))
	if err != nil {
		return nil, err
	}

	return &AggregateResponse{Message: r}, nil
}

func (o *overlayService) SendCollectionRequest(ctx context.Context, in *CollectionRequest) (*CollectionResponse, error) {
	replies, err := o.Overlay.sendCollect(in.GetId(), in)
	if err != nil {
		return nil, err
	}

	return &CollectionResponse{Message: replies}, nil
}
