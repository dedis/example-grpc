generate:
	go generate ./...

test: generate
	go test -count=1 ./...

.DEFAULT_GOAL := run
