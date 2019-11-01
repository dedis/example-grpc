package skipchain

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.dedis.ch/example-grpc/overlay"
	"go.dedis.ch/kyber/v4/sign/bdn"
)

func TestSkipchain_Signing(t *testing.T) {
	n := 5
	servers := make([]*overlay.Overlay, n)
	skipchains := make([]*Skipchain, n)
	for i := range servers {
		o := overlay.NewOverlay("localhost:0")
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

	sig, err := skipchains[0].Sign([]byte("deadbeef"), roster)
	require.NoError(t, err)
	require.NotNil(t, sig)

	aggPub, err := skipchains[0].GetAggregatePublicKey(roster)
	require.NoError(t, err)
	require.NoError(t, bdn.Verify(suite, aggPub, []byte("deadbeef"), sig))
}
