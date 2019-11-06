generate:
	go generate ./...

test: generate
	go test -count=1 ./...

# If generate fails because protoc-gen-go is not installed, use this to install it.
protoc-gen-go:
	go get -u github.com/golang/protobuf/protoc-gen-go

.DEFAULT_GOAL := run
