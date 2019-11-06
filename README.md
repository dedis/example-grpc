# Onet using gRPC

This repository contains an example of an network overlay that is using gRPC
for the network layer.

## How to

Install the required protoc-gen-go plugin with:
```
make protoc-gen-go
```

Generate code from proto files:
```
make generate
```

Run the tests:
```
make test
```

## Aggregation

The overlay provides an aggregation protocol using a tree-based topology. When
defined, the protocol will take care of gathering the public identity of each
node.

## Architecture

The overlay is built on top of gRPC. It provides features to create protocols
based on multiple interfaces. The first available protocol is the aggregation
that will contact a roster of nodes and ask them for a specific value defined
by the protocol implementation. It will then gather and merge the responses to
the root.

TLS is configured with self-signed certificates which means that the public key
needs to be passed around the roster so that each node has the certificate of
the neighbours it wants to communicate with. TLS is configured to ask for a
certificate but anonymous client can still make requests. Some logic requires
a client authentication like for the protocols.

A wrapper is used to enable Web API compatibility and the requests are made on
the same port than P2P requests but it is limited according to [gRPC-web](https://github.com/improbable-eng/grpc-web).

### Aggregation with identity

The aggregation protocol will provide the protocol identities when the dedicated
interface is implemented. It is up to the protocol implementation to cache them.
The overlay insures that all the identities required are present before calling
`Process`.

TODO: what to do when a node is offline ? Continue with a truncated
roster ?
