package overlay

import (
	context "context"
	"errors"
	fmt "fmt"
	"time"

	proto "github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
)

// Aggregation is the interface to implement to register a protocol that
// will contact all the nodes and aggregate their replies.
type Aggregation interface {
	Process(proto.Message, []proto.Message) (proto.Message, error)
}

// AggregationWithIdentity is the interface to implement to register a
// protocol that will provide the identity of every other involved nodes.
type AggregationWithIdentity interface {
	Aggregation

	Identity() (proto.Message, error)

	StoreIdentities(map[string]proto.Message)
}

// Aggregate starts a new aggregation protocol that will gather a response from
// every node in the roster. The message contains the parameter of the protocol.
func (o *Overlay) Aggregate(name string, ro Roster, in proto.Message) (proto.Message, error) {
	agg := o.aggregators[name]
	if agg == nil {
		return nil, errors.New("aggregation not found")
	}

	root, err := o.GetPeer()
	if err != nil {
		return nil, fmt.Errorf("couldn't get the root: %v", err)
	}

	idents := []*Identity{}
	if aggI, ok := agg.(AggregationWithIdentity); ok {
		// The procool implements the identity support so the leader
		// will gather the identities and send them back with the
		// aggregation request.
		// TODO: improvement to trigger this only when necessary.
		req := &IdentityRequest{
			Protocol: name,
			Tree:     ro.makeTree(root),
		}

		// Each protocol can implement its own public identity.
		ident, err := aggI.Identity()
		if err != nil {
			return nil, fmt.Errorf("couldn't generate the identity: %v", err)
		}

		// Gather the different identities involved.
		idents, err = o.sendIdentityRequest(req, ident)
		if err != nil {
			return nil, fmt.Errorf("couldn't request the identities: %v", err)
		}
	}

	msg, err := ptypes.MarshalAny(in)
	if err != nil {
		return nil, fmt.Errorf("couldn't marshal: %v", err)
	}

	req := &AggregateRequest{
		Protocol:   name,
		Tree:       ro.makeTree(root),
		Identities: idents,
		Message:    msg,
	}

	replies, err := o.sendAggregateRequest(req, agg)
	if err != nil {
		return nil, fmt.Errorf("couldn't aggregate: %v", err)
	}

	return agg.Process(in, replies)
}

func (o *Overlay) sendIdentityRequest(msg *IdentityRequest, ident proto.Message) ([]*Identity, error) {
	children := msg.GetTree().getChildren(o.listener.Addr().String())
	idents := make([]*Identity, 0, len(children))
	for _, conn := range o.getConnections(children) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		client := NewOverlayClient(conn)
		defer conn.Close()

		resp, err := client.Identity(ctx, msg)
		if err != nil {
			return nil, err
		}

		idents = append(idents, resp.GetIdentities()...)
	}

	value, err := ptypes.MarshalAny(ident)
	if err != nil {
		return nil, err
	}

	idents = append(idents, &Identity{
		Addr:      o.listener.Addr().String(),
		Signature: []byte{},
		Value:     value,
	})

	return idents, nil
}

// Identity is the handler of identity requests. It will contact the children if
// any and then send back the list of known identities.
func (o *overlayService) Identity(ctx context.Context, in *IdentityRequest) (*IdentityResponse, error) {
	agg := o.Overlay.aggregators[in.GetProtocol()]
	if agg == nil {
		return nil, errors.New("aggregation not found")
	}

	ident, err := agg.(AggregationWithIdentity).Identity()
	if err != nil {
		return nil, fmt.Errorf("couldn't generate the identity: %v", err)
	}

	replies, err := o.Overlay.sendIdentityRequest(in, ident)
	if err != nil {
		return nil, fmt.Errorf("couldn't send the identity request: %v", err)
	}

	return &IdentityResponse{Identities: replies}, nil
}

func (o *Overlay) sendAggregateRequest(msg *AggregateRequest, agg Aggregation) ([]proto.Message, error) {
	if aggI, ok := agg.(AggregationWithIdentity); ok {
		store := make(map[string]proto.Message)
		for _, ident := range msg.GetIdentities() {
			var da ptypes.DynamicAny
			err := ptypes.UnmarshalAny(ident.GetValue(), &da)
			if err != nil {
				return nil, err
			}

			store[ident.GetAddr()] = da.Message
		}

		// Make the identities provided by the protocol available in the
		// skipchain engine.
		// TODO: improvement to request missing identities.
		aggI.StoreIdentities(store)
	}

	children := msg.GetTree().getChildren(o.listener.Addr().String())
	replies := make([]proto.Message, 0, len(children))
	for _, conn := range o.getConnections(children) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		client := NewOverlayClient(conn)
		defer conn.Close()

		resp, err := client.Aggregate(ctx, msg)
		if err != nil {
			return nil, err
		}

		var da ptypes.DynamicAny
		err = ptypes.UnmarshalAny(resp.GetMessage(), &da)
		if err != nil {
			return nil, err
		}

		replies = append(replies, da.Message)
	}

	return replies, nil
}

// Aggregate is the handler for aggregation requests.
func (o *overlayService) Aggregate(ctx context.Context, in *AggregateRequest) (*AggregateResponse, error) {
	_, err := o.Overlay.peerFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("couldn't get client info: %v", err)
	}

	agg := o.Overlay.aggregators[in.GetProtocol()]
	if agg == nil {
		return nil, errors.New("aggregation not found")
	}

	replies, err := o.Overlay.sendAggregateRequest(in, agg)
	if err != nil {
		return nil, fmt.Errorf("couldn't send the aggregate request: %v", err)
	}

	var da ptypes.DynamicAny
	err = ptypes.UnmarshalAny(in.GetMessage(), &da)
	if err != nil {
		return nil, fmt.Errorf("couldn't unmarshal the reply: %v", err)
	}

	res, err := agg.Process(da.Message, replies)
	if err != nil {
		return nil, fmt.Errorf("couldn't process: %v", err)
	}

	r, err := ptypes.MarshalAny(res)
	if err != nil {
		return nil, fmt.Errorf("couldn't marshal: %v", err)
	}

	return &AggregateResponse{Message: r}, nil
}
