package overlay

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"io/ioutil"
	"net/http"
	"testing"

	proto "github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/http2"
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
	require.NoError(t, servers[2].GracefulStop())

	agg, err := servers[0].Aggregate(testAggregationName, roster, &TestMessage{Value: 0})
	require.NoError(t, err)
	require.Equal(t, int64(n-1), agg.(*TestMessage).GetValue())
}

// Simulate a web request to make sure the wrapper is working.
func TestOverlay_WebRequest(t *testing.T) {
	o := NewOverlay("127.0.0.1:2000")
	go func() {
		require.NoError(t, o.Serve())
	}()
	<-o.StartChan

	pool := x509.NewCertPool()
	pool.AddCert(o.cert.Leaf)

	tr := &http2.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs: pool,
		},
	}
	client := &http.Client{Transport: tr}

	msg := "Hello World!"
	buf, err := proto.Marshal(&EchoMessage{Value: msg})
	require.NoError(t, err)

	preamble := []byte{0, 0, 0, 0, 0}
	binary.BigEndian.PutUint32(preamble[1:], uint32(len(buf)))

	writer := new(bytes.Buffer)
	writer.Write(preamble)
	writer.Write(buf)

	resp, err := client.Post("https://127.0.0.1:2000/overlay.Overlay/echo", "application/grpc-web", writer)
	require.NoError(t, err)
	defer resp.Body.Close()

	contents, _ := ioutil.ReadAll(resp.Body)
	ret := &EchoMessage{}
	err = proto.Unmarshal(contents[5:5+len(buf)], ret)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode)
	require.Equal(t, msg, ret.GetValue())
}
