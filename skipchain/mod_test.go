package skipchain

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/dedis/example-grpc/overlay"
	"github.com/stretchr/testify/require"
	"go.dedis.ch/kyber/v4/sign/bdn"
)

func TestSkipchain_Signing(t *testing.T) {
	n := 5
	servers := make([]*overlay.Overlay, n)
	skipchains := make([]*Skipchain, n)
	for i := range servers {
		o := overlay.NewOverlay(fmt.Sprintf("overlay-%d", i), "localhost:0")
		servers[i] = o
		skipchains[i] = NewSkipchain(o)

		go func() {
			err := o.Serve()
			require.NoError(t, err)
		}()

		<-o.StartChan
	}

	roster := make([]overlay.Peer, n)
	for i, srv := range servers {
		curr := srv.GetPeer()
		roster[i] = curr
		for _, other := range servers[i+1:] {
			peer := other.GetPeer()
			require.NoError(t, srv.AddNeighbour(peer))
			require.NoError(t, other.AddNeighbour(curr))
		}
	}

	ctx := context.Background()
	sig, err := skipchains[0].Sign(ctx, []byte("deadbeef"), roster)
	require.NoError(t, err)
	require.NotNil(t, sig)

	aggPub, err := skipchains[0].GetAggregatePublicKey(roster)
	require.NoError(t, err)
	require.NoError(t, bdn.Verify(suite, aggPub, []byte("deadbeef"), sig))

	// It should still work with one node down.
	servers[2].GracefulStop()

	sig, err = skipchains[1].Sign(ctx, []byte("deadbeef"), roster)
	require.NoError(t, err)
	require.NotNil(t, sig)

	// wait for spans to reach
	time.Sleep(2 * time.Second)
}
