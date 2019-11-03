package skipchain

import (
	fmt "fmt"

	"github.com/dedis/example-grpc/overlay"
	"go.dedis.ch/kyber/v4"
	"go.dedis.ch/kyber/v4/pairing"
	"go.dedis.ch/kyber/v4/sign"
	"go.dedis.ch/kyber/v4/sign/bdn"
	"go.dedis.ch/kyber/v4/util/key"
)

//go:generate protoc -I ./ --go_out=plugins=grpc:./ ./skipchain.proto

var suite = pairing.NewSuiteBn256()

// Skipchain is the engine that provides the necessary functions to manage a blockchain.
type Skipchain struct {
	overlay    *overlay.Overlay
	keyPair    *key.Pair
	identities map[string]kyber.Point
}

// NewSkipchain returns an instance of the engine.
func NewSkipchain(overlay *overlay.Overlay) *Skipchain {
	kp := key.NewKeyPair(suite)

	skipchain := &Skipchain{
		overlay:    overlay,
		keyPair:    kp,
		identities: make(map[string]kyber.Point),
	}

	overlay.RegisterAggregation(bdnCoSi, newCosiAggregate(skipchain))

	return skipchain
}

// GetPublicKey returns the public key for the node.
func (sc *Skipchain) GetPublicKey() kyber.Point {
	return sc.keyPair.Public
}

// GetPublicKeys return the list of public keys available.
func (sc *Skipchain) GetPublicKeys(ro overlay.Roster) []kyber.Point {
	pubs := make([]kyber.Point, len(ro))
	for i, key := range ro {
		pubs[i] = sc.identities[key.Address]
	}

	return pubs
}

// GetAggregatePublicKey returns the aggregate of the known
// public keys.
func (sc *Skipchain) GetAggregatePublicKey(ro overlay.Roster) (kyber.Point, error) {
	pubs := sc.GetPublicKeys(ro)
	mask, _ := sign.NewMask(suite, pubs, nil)
	for i := range pubs {
		err := mask.SetBit(i, true)
		if err != nil {
			return nil, fmt.Errorf("couldn't create the mask: %v", err)
		}
	}

	return bdn.AggregatePublicKeys(suite, mask)
}

// Sign requests a signature by the roster.
func (sc *Skipchain) Sign(msg []byte, ro overlay.Roster) ([]byte, error) {
	agg, err := sc.overlay.Aggregate(bdnCoSi, ro, &SigningRequest{Message: msg})
	if err != nil {
		return nil, fmt.Errorf("couldn't aggregate: %v", err)
	}

	resp := agg.(*SigningResponse)

	return resp.GetSignature(), nil
}
