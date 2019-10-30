# Onet using gRPC

This repository contains an example of an network overlay that is using gRPC
for the network layer.

## How to

Servers can be run using the command:
```
make
```

And a client request can be made with:
```
go run ./client/client.go
```

## Architecture

The idea is to have services that can use the overlay to run protocols like
the aggregation or the collection. The service provides only the processor
for the protocol but the overlay takes care of sending the messages around.

Clients would only need to make requests to the services.

```
  ----------------------------------------------------        -----------------------
  | OVERLAY                                          |        | Service Count       |
  |                                                  |--------|                     |
  | ----------------------  ------------------------ |        |                     |
  | | AGGREGATION        |  | COLLECTION           | |        -----------------------
  | |   PROTOCOL         |  |    PROTOCOL          | |     --------          |
  | |                    |  |                      | |-----| ...  |          |
  | ----------------------  ------------------------ |     --------          |
  ----------------------------------------------------          |            |
                                                                -----------Client


```