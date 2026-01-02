APP_NAME := go-mcp-gitlab
VERSION := 1.0.0

.PHONY: build build-all test clean install

build:
	go build -ldflags "-X main.Version=$(VERSION)" -o $(APP_NAME) .

build-linux:
	GOOS=linux GOARCH=amd64 go build -ldflags "-X main.Version=$(VERSION)" -o $(APP_NAME)-linux-amd64 .

build-darwin:
	GOOS=darwin GOARCH=amd64 go build -ldflags "-X main.Version=$(VERSION)" -o $(APP_NAME)-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build -ldflags "-X main.Version=$(VERSION)" -o $(APP_NAME)-darwin-arm64 .

build-windows:
	GOOS=windows GOARCH=amd64 go build -ldflags "-X main.Version=$(VERSION)" -o $(APP_NAME)-windows-amd64.exe .

build-all: build-linux build-darwin build-windows

test:
	go test -v ./...

clean:
	rm -f $(APP_NAME) $(APP_NAME)-*

install:
	go install -ldflags "-X main.Version=$(VERSION)" .

fmt:
	go fmt ./...

vet:
	go vet ./...
