package skipchain

import (
	fmt "fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"go.dedis.ch/example-grpc/overlay"
	"go.dedis.ch/kyber/v4/sign/bdn"
)

func TestSkipchain_Signing(t *testing.T) {
	n := 5
	servers := make([]*overlay.Overlay, n)
	skipchains := make([]*Skipchain, n)
	roster := make([]overlay.Peer, n)
	for i := range servers {
		o := overlay.NewOverlay(fmt.Sprintf("localhost:310%d", i))
		servers[i] = o
		skipchains[i] = NewSkipchain(o)
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

	sig, err := skipchains[0].Sign([]byte("deadbeef"), roster)
	require.NoError(t, err)
	require.NotNil(t, sig)

	aggPub, err := skipchains[0].GetAggregatePublicKey()
	require.NoError(t, err)
	require.NoError(t, bdn.Verify(suite, aggPub, []byte("deadbeef"), sig))
}
