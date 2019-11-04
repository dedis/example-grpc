package overlay

import (
	context "context"
	"errors"
	fmt "fmt"

	proto "github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
)

// AggregationContext is provided to the protocol functions so that
// useful information can be retrieved from the context.
type AggregationContext interface {
	GetMessage() proto.Message
	GetRoster() Roster
}

// SimpleAggregationContext is a context implementation that will use
// the overlay to extract the information.
type SimpleAggregationContext struct {
	overlay *Overlay
	request *PropagationRequest
	message proto.Message
}

func newSimpleAggregationContext(req *PropagationRequest, o *Overlay) (SimpleAggregationContext, error) {
	var da ptypes.DynamicAny
	err := ptypes.UnmarshalAny(req.GetMessage(), &da)
	if err != nil {
		return SimpleAggregationContext{}, fmt.Errorf("couldn't unmarshal message: %v", err)
	}

	return SimpleAggregationContext{
		request: req,
		message: da.Message,
		overlay: o,
	}, nil
}

// GetMessage returns the request message for this aggregation.
func (sac SimpleAggregationContext) GetMessage() proto.Message {
	return sac.message
}

// GetRoster returns the roster for this aggregation.
func (sac SimpleAggregationContext) GetRoster() Roster {
	roster := make(Roster, len(sac.request.GetTree().GetAddresses()))
	curr := sac.overlay.GetPeer()

	for i, addr := range sac.request.GetTree().GetAddresses() {
		if addr == curr.Address {
			roster[i] = curr
		} else {
			roster[i] = sac.overlay.neighbours[addr]
		}
	}

	return roster
}

// Aggregation is the interface to implement to register a protocol that
// will contact all the nodes and aggregate their replies.
type Aggregation interface {
	Process(AggregationContext, []proto.Message) (proto.Message, error)
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

	root := o.GetPeer()

	idents := []*Identity{}
	if aggI, ok := agg.(AggregationWithIdentity); ok {
		// The procool implements the identity support so the leader
		// will gather the identities and send them back with the
		// aggregation request.
		// TODO: improvement to trigger this only when necessary.
		req := &PropagationRequest{
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

	req := &PropagationRequest{
		Protocol:   name,
		Tree:       ro.makeTree(root),
		Identities: idents,
		Message:    msg,
	}

	res, err := o.sendAggregateRequest(req, agg)
	if err != nil {
		return nil, fmt.Errorf("couldn't aggregate: %v", err)
	}

	return res, nil
}

func (o *Overlay) sendIdentityRequest(in *PropagationRequest, ident proto.Message) ([]*Identity, error) {
	replies, err := o.Propagate(in, o.listener.Addr().String(), func(cl OverlayClient) PropagateFn {
		return cl.Identity
	})

	idents := make([]*Identity, 0)
	for _, reply := range replies {
		idents = append(idents, reply.(*IdentityResponse).GetIdentities()...)
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

func (o *Overlay) sendAggregateRequest(msg *PropagationRequest, agg Aggregation) (proto.Message, error) {
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

	replies, err := o.Propagate(msg, o.listener.Addr().String(), func(cl OverlayClient) PropagateFn {
		return cl.Aggregate
	})
	if err != nil {
		return nil, fmt.Errorf("couldn't aggregate the children: %v", err)
	}

	aggCtx, err := newSimpleAggregationContext(msg, o)
	if err != nil {
		return nil, fmt.Errorf("couldn't create the context: %v", err)
	}

	res, err := agg.Process(aggCtx, replies)
	if err != nil {
		return nil, fmt.Errorf("couldn't process: %v", err)
	}

	return res, nil
}

// Identity is the handler of identity requests. It will contact the children if
// any and then send back the list of known identities.
func (o *overlayService) Identity(ctx context.Context, in *PropagationRequest) (*PropagationResponse, error) {
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

	buf, err := ptypes.MarshalAny(&IdentityResponse{Identities: replies})
	if err != nil {
		return nil, fmt.Errorf("couldn't marshal the response: %v", err)
	}

	return &PropagationResponse{Message: buf}, nil
}

// Aggregate is the handler for aggregation requests.
func (o *overlayService) Aggregate(ctx context.Context, in *PropagationRequest) (*PropagationResponse, error) {
	_, err := o.Overlay.peerFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("couldn't get client info: %v", err)
	}

	agg := o.Overlay.aggregators[in.GetProtocol()]
	if agg == nil {
		return nil, errors.New("aggregation not found")
	}

	res, err := o.Overlay.sendAggregateRequest(in, agg)
	if err != nil {
		return nil, fmt.Errorf("couldn't send the aggregate request: %v", err)
	}

	r, err := ptypes.MarshalAny(res)
	if err != nil {
		return nil, fmt.Errorf("couldn't marshal: %v", err)
	}

	return &PropagationResponse{Message: r}, nil
}
