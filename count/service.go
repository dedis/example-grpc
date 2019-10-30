package count // import "go.dedis.ch/eonet/count"

import (
	context "context"

	proto "github.com/golang/protobuf/proto"
	"go.dedis.ch/example-grpc/overlay"
)

//go:generate protoc -I ./ --go_out=plugins=grpc:./ ./count.proto

const (
	// AggregationProtocolName is the name of the count aggregation.
	AggregationProtocolName = "CountAggregation"

	// CollectionProtocolName is the name of the count collection.
	CollectionProtocolName = "CountCollection"
)

// AggregationProtocol implements a aggregatation protocol to count the number of
// online nodes in the roster.
type AggregationProtocol struct{}

// Announce returns the announcement message that the root will send to
// its children.
func (a *AggregationProtocol) Announce() proto.Message {
	return &Counter{Value: 0}
}

// Process takes the replies of the children and create the aggregate that
// will be sent to the parent.
func (a *AggregationProtocol) Process(replies []proto.Message) proto.Message {
	sum := int64(0)
	for _, r := range replies {
		msg := r.(*Counter)
		sum += msg.GetValue()
	}

	return &Counter{Value: sum + 1}
}

// Spread takes the announcement message and returns the new one that will be
// spread to the children.
func (a *AggregationProtocol) Spread(in proto.Message) proto.Message {
	return in
}

type CollectionProtocol struct {
	Peer Peer
}

func (c *CollectionProtocol) Prepare() proto.Message {
	return &c.Peer
}

// Service is the implementation of the RPC count service.
type Service struct {
	UnimplementedCountServer
	*overlay.Overlay
}

// NewService is blabla
func NewService(overlay *overlay.Overlay) *Service {
	peer := overlay.GetPeer()

	overlay.RegisterAggregation(AggregationProtocolName, &AggregationProtocol{})
	overlay.RegisterCollection(CollectionProtocolName, &CollectionProtocol{
		Peer: Peer{
			Port:        peer.Port,
			Certificate: peer.Certificate.Raw,
		},
	})

	return &Service{Overlay: overlay}
}

// Count is
func (s *Service) Count(ctx context.Context, in *CountRequest) (*CountResponse, error) {
	collect, err := s.Collect(CollectionProtocolName)
	if err != nil {
		return nil, err
	}

	certificates := make([][]byte, len(collect))
	for i, c := range collect {
		certificates[i] = c.(*Peer).Certificate
	}

	agg, err := s.Aggregate(AggregationProtocolName)
	if err != nil {
		return nil, err
	}

	counter := agg.(*Counter)

	return &CountResponse{Value: counter.GetValue(), Certificates: certificates}, nil
}
