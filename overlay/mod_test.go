package overlay

import (
	"testing"

	proto "github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/require"
)

const testAggregationName = "go.dedis.ch/example-grpc/overlay.testAggregation"

type testAggregation struct{}

// Process takes the replies of the children and create the aggregate that
// will be sent to the parent.
func (a testAggregation) Process(ctx AggregationContext, replies []proto.Message) (proto.Message, error) {
	sum := int64(0)
	for _, r := range replies {
		msg := r.(*TestMessage)
		sum += msg.GetValue()
	}

	return &TestMessage{Value: sum + 1}, nil
}

func TestOverlay_SimpleAggregation(t *testing.T) {
	n := 5
	servers := make([]*Overlay, n)
	for i := range servers {
		o := NewOverlay("localhost:0")
		o.RegisterAggregation(testAggregationName, testAggregation{})
		servers[i] = o

		go func() {
			err := o.Serve()
			require.NoError(t, err)
		}()

		<-o.StartChan
	}

	roster := make([]Peer, n)
	for i, srv := range servers {
		curr := srv.GetPeer()
		roster[i] = curr
		for _, other := range servers[i+1:] {
			peer := other.GetPeer()
			require.NoError(t, srv.AddNeighbour(peer))
			require.NoError(t, other.AddNeighbour(curr))
		}
	}

	// Kill one server to test the resilience.
	servers[2].GracefulStop()

	agg, err := servers[0].Aggregate(testAggregationName, roster, &TestMessage{Value: 0})
	require.NoError(t, err)
	require.Equal(t, int64(n-1), agg.(*TestMessage).GetValue())
}
