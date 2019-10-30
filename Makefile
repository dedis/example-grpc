generate:
	go generate ./...

run: generate
	go run ./server/server.go

.DEFAULT_GOAL := run
