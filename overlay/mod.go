package overlay

import (
	context "context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"errors"
	fmt "fmt"
	"log"
	"math/big"
	"net"
	"time"

	proto "github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
)

//go:generate protoc -I ./ --go_out=plugins=grpc:./ ./overlay.proto

// Peer is a public identity for a given node.
type Peer struct {
	Address     string
	Certificate *x509.Certificate
}

// Roster is a set of peers that will work together
// to execute protocols.
type Roster []Peer

func (ro Roster) makeTree(root Peer) *Tree {
	t := &Tree{
		Addresses: make([]string, len(ro)),
		K:         2,
	}

	for i, peer := range ro {
		if peer.Address == root.Address {
			t.Addresses[0] = root.Address
		} else if t.Addresses[0] == "" {
			t.Addresses[i+1] = peer.Address
		} else {
			t.Addresses[i] = peer.Address
		}
	}

	return t
}

func (ro Roster) makeExceptTree(root Peer, addrs map[string]proto.Message) *Tree {
	ro2 := make(Roster, 0, len(ro))
	for _, p := range ro {
		if _, ok := addrs[p.Address]; !ok {
			ro2 = append(ro2, p)
		}
	}

	return ro2.makeTree(root)
}

// PropagateFn is a handler for a specific type of propagation. For instance,
// an aggregation will propagate a message and retrieve a response from the
// participants.
type PropagateFn func(context.Context, *PropagationRequest, ...grpc.CallOption) (*PropagationResponse, error)

// PropagateFnGenerator returns the propagation function from the client.
type PropagateFnGenerator func(client OverlayClient) PropagateFn

// Propagator enables the support of propagation protocols.
type Propagator interface {
	Propagate(in *PropagationRequest, addr string, fn PropagateFnGenerator) ([]proto.Message, error)
}

// Overlay is the network abstraction to communicate with the roster.
type Overlay struct {
	*grpc.Server

	Propagator

	cert      *tls.Certificate
	addr      string
	listener  net.Listener
	StartChan chan struct{}

	// neighbours contains the certificate and details about known peers.
	neighbours map[string]Peer

	aggregators map[string]Aggregation
}

// NewOverlay returns a new overlay.
func NewOverlay(addr string) *Overlay {
	cert := makeCertificate()

	ta := credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{*cert},
		ClientAuth:   tls.RequireAnyClientCert,
	})
	srv := grpc.NewServer(grpc.Creds(ta))

	overlay := &Overlay{
		Server:      srv,
		cert:        cert,
		addr:        addr,
		listener:    nil,
		StartChan:   make(chan struct{}),
		neighbours:  make(map[string]Peer),
		aggregators: make(map[string]Aggregation),
	}

	RegisterOverlayServer(srv, &overlayService{Overlay: overlay})

	return overlay
}

// GetPeer makes and returns the public identity of the node.
func (o *Overlay) GetPeer() Peer {
	if o.listener == nil {
		panic("server not started")
	}

	return Peer{
		Address:     o.listener.Addr().String(),
		Certificate: o.cert.Leaf,
	}
}

// AddNeighbour inserts the peer in the list of known peers
// which will allow to communicate with it.
// TODO: how to pass certs at runtime.
func (o *Overlay) AddNeighbour(peer Peer) error {
	o.neighbours[peer.Address] = peer
	return nil
}

// RegisterAggregation stores the aggregation using the name
// as a key.
func (o *Overlay) RegisterAggregation(name string, impl Aggregation) {
	o.aggregators[name] = impl
}

// Propagate takes care of spreading the incoming request to the node's
// children, processing their responses and send them to the parent.
func (o *Overlay) Propagate(in *PropagationRequest, addr string, fn PropagateFnGenerator) ([]proto.Message, error) {
	children := in.GetTree().getChildren(addr)
	replies := make([]proto.Message, 0, len(children))
	for i, conn := range o.getConnections(children) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		client := NewOverlayClient(conn)
		defer conn.Close()

		resp, err := fn(client)(ctx, in)
		if err != nil {
			// If the client is not responsive, we contact its children directly.
			log.Printf("Couldn't contact [%s]. Taking care of its children.\n", children[i])
			childReplies, err := o.Propagate(in, children[i], fn)
			if err != nil {
				return nil, err
			}

			replies = append(replies, childReplies...)
		} else {
			var da ptypes.DynamicAny
			err = ptypes.UnmarshalAny(resp.GetMessage(), &da)
			if err != nil {
				return nil, err
			}

			replies = append(replies, da.Message)
		}
	}

	return replies, nil
}

func (o *Overlay) getConnection(addr string) (*grpc.ClientConn, error) {
	neighbour, ok := o.neighbours[addr]
	if !ok {
		return nil, fmt.Errorf("couldn't find neighbour [%s]", addr)
	}

	pool := x509.NewCertPool()
	pool.AddCert(neighbour.Certificate)

	ta := credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{*o.cert},
		RootCAs:      pool,
	})

	// Connecting using TLS and the distant server certificate as the root.
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(ta))
	if err != nil {
		return nil, fmt.Errorf("couldn't dial: %v", err)
	}

	return conn, nil
}

func (o *Overlay) getConnections(addrs []string) []*grpc.ClientConn {
	peers := make([]*grpc.ClientConn, 0, len(addrs))
	for _, addr := range addrs {
		conn, err := o.getConnection(addr)
		if err != nil {
			log.Printf("couldn't open the connection: %+v", err)
		} else {
			// Add the neighbour only if we can dial.
			peers = append(peers, conn)
		}
	}
	return peers
}

// Serve starts the overlay to listen on the address.
func (o *Overlay) Serve() error {
	lis, err := net.Listen("tcp4", o.addr)
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}

	o.listener = lis

	log.Printf("Server [%v] is starting...\n", lis.Addr())
	close(o.StartChan)

	if err := o.Server.Serve(lis); err != nil {
		return fmt.Errorf("failed to serve: %v", err)
	}

	log.Printf("Server [%v] has stopped...\n", lis.Addr())

	return nil
}

func (o *Overlay) peerFromContext(ctx context.Context) (Peer, error) {
	client, ok := peer.FromContext(ctx)
	if !ok {
		return Peer{}, errors.New("unknown client")
	}

	info := client.AuthInfo.(credentials.TLSInfo)
	for _, cert := range info.State.PeerCertificates {
		// TODO: verify the client certificate as we only want authentication
		// for protocols so we don't enforce the client certificate to be correct.
		for _, neighbour := range o.neighbours {
			if neighbour.Certificate.Equal(cert) {
				return neighbour, nil
			}
		}
	}

	return Peer{}, errors.New("unauthenticated client")
}

// gRPC service for the overlay.
type overlayService struct {
	UnimplementedOverlayServer

	Overlay *Overlay
}

func makeCertificate() *tls.Certificate {
	priv, err := ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
	if err != nil {
		log.Fatalf("Couldn't generate the private key: %+v", err)
	}

	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(time.Hour * 24 * 180),

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	buf, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &priv.PublicKey, priv)
	if err != nil {
		log.Fatalf("Couldn't create the certificate: %+v", err)
	}

	cert, err := x509.ParseCertificate(buf)
	if err != nil {
		log.Fatalf("Couldn't parse the certificate: %+v", err)
	}

	return &tls.Certificate{
		Certificate: [][]byte{buf},
		PrivateKey:  priv,
		Leaf:        cert,
	}
}
