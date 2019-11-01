package overlay

import (
	fmt "fmt"
	"testing"

	proto "github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/require"
)

const testAggregationName = "go.dedis.ch/example-grpc/overlay.testAggregation"

type testAggregation struct{}

// Process takes the replies of the children and create the aggregate that
// will be sent to the parent.
func (a testAggregation) Process(ann proto.Message, replies []proto.Message) (proto.Message, error) {
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
	roster := make([]Peer, 5)
	for i := range servers {
		o := NewOverlay(fmt.Sprintf("localhost:300%d", i))
		o.RegisterAggregation(testAggregationName, testAggregation{})
		servers[i] = o
		roster[i] = o.GetPeer()

		if i > 0 {
			for _, srv := range servers[:i] {
				require.NoError(t, srv.AddNeighbour(o.GetPeer()))
				require.NoError(t, o.AddNeighbour(srv.GetPeer()))
			}
		}

		go func() {
			err := o.Serve()
			require.NoError(t, err)
		}()
	}

	for _, srv := range servers {
		<-srv.StartChan
	}

	agg, err := servers[0].Aggregate(testAggregationName, roster, &TestMessage{Value: 0})
	require.NoError(t, err)
	require.Equal(t, int64(5), agg.(*TestMessage).GetValue())
}
