generate:
	go generate ./...

test: generate
	go test -count=1 -v ./...

.DEFAULT_GOAL := run
