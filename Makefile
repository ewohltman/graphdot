.PHONY: lint build install

lint:
	golangci-lint run ./...

build:
	go build graphdot.go

install:
	go install graphdot.go
