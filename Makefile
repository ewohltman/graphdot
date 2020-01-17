.PHONY: build install

build:
	go build -o graphdot graphdot.go

install:
	go install graphdot.go
