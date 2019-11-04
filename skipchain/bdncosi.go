package skipchain

import (
	fmt "fmt"

	"github.com/dedis/example-grpc/overlay"
	"github.com/golang/protobuf/proto"
	"go.dedis.ch/kyber/v4/sign"
	"go.dedis.ch/kyber/v4/sign/bdn"
	"go.dedis.ch/kyber/v4/sign/bls"
)

const (
	bdnCoSi = "go.dedis.ch/example-grpc/skipchain.BdnCosi"
)

type cosiAggregate struct {
	skipchain *Skipchain
}

func newCosiAggregate(skipchain *Skipchain) *cosiAggregate {
	return &cosiAggregate{skipchain: skipchain}
}

// Process will create the signature of the current node and
// append it to the children's signatures and return the
// aggregate to the parent.
func (cosi *cosiAggregate) Process(ctx overlay.AggregationContext, replies []proto.Message) (proto.Message, error) {
	req := ctx.GetMessage().(*SigningRequest)
	sig, err := bdn.Sign(suite, cosi.skipchain.keyPair.Private, req.GetMessage())
	if err != nil {
		return nil, fmt.Errorf("fail to sign the message: %v", err)
	}

	mask, err := sign.NewMask(suite, cosi.skipchain.GetPublicKeys(ctx.GetRoster()), cosi.skipchain.GetPublicKey())
	if err != nil {
		return nil, fmt.Errorf("couldn't create the mask: %v", err)
	}

	sigPt, err := bdn.AggregateSignatures(suite, [][]byte{sig}, mask)
	if err != nil {
		return nil, err
	}
	sig, err = sigPt.MarshalBinary()
	if err != nil {
		return nil, err
	}

	sigs := make([][]byte, len(replies)+1)
	sigs[0] = sig
	for i, reply := range replies {
		sigs[i+1] = reply.(*SigningResponse).GetSignature()
	}

	agg, err := bls.AggregateSignatures(suite, sigs...)
	if err != nil {
		return nil, err
	}

	return &SigningResponse{Signature: agg}, nil
}

// Identity returns the public identity for the node.
func (cosi *cosiAggregate) Identity() (proto.Message, error) {
	buf, err := cosi.skipchain.GetPublicKey().MarshalBinary()
	if err != nil {
		return nil, err
	}

	return &BdnIdentity{PublicKey: buf}, nil
}

// StoreIdentities takes the identities in parameter and store them
// to the skipchain engine so that they can be fetched later.
func (cosi *cosiAggregate) StoreIdentities(idents map[string]proto.Message) {
	for k, v := range idents {
		point := suite.G2().Point()
		err := point.UnmarshalBinary(v.(*BdnIdentity).PublicKey)
		if err != nil {
			panic(err)
		}

		cosi.skipchain.identities[k] = point
	}
}

func (cosi *cosiAggregate) GetIdentities() map[string]proto.Message {
	idents := make(map[string]proto.Message)
	for k, v := range cosi.skipchain.identities {
		bb, _ := v.MarshalBinary()

		idents[k] = &BdnIdentity{
			PublicKey: bb,
		}
	}

	return idents
}
