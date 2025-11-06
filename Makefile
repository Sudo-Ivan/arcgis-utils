BINARY_NAME=arcgis-utils
CMD_PATH=./cmd/arcgis-utils
LDFLAGS=-s -w
LDFLAGS_DEBUG=
GOOS=$(shell go env GOOS)
GOARCH=$(shell go env GOARCH)

.PHONY: all build debug clean

all: build

build:
	mkdir -p bin
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build -ldflags="$(LDFLAGS)" -o bin/$(BINARY_NAME) $(CMD_PATH)

debug:
	mkdir -p bin
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build -ldflags="$(LDFLAGS_DEBUG)" -o bin/$(BINARY_NAME) $(CMD_PATH)

linux-amd64:
	mkdir -p bin
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o bin/$(BINARY_NAME)-linux-amd64 $(CMD_PATH)

linux-386:
	mkdir -p bin
	CGO_ENABLED=0 GOOS=linux GOARCH=386 go build -ldflags="$(LDFLAGS)" -o bin/$(BINARY_NAME)-linux-386 $(CMD_PATH)

linux-arm64:
	mkdir -p bin
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o bin/$(BINARY_NAME)-linux-arm64 $(CMD_PATH)

linux-armv6:
	mkdir -p bin
	CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=6 go build -ldflags="$(LDFLAGS)" -o bin/$(BINARY_NAME)-linux-armv6 $(CMD_PATH)

linux-armv7:
	mkdir -p bin
	CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go build -ldflags="$(LDFLAGS)" -o bin/$(BINARY_NAME)-linux-armv7 $(CMD_PATH)

windows-amd64:
	mkdir -p bin
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o bin/$(BINARY_NAME)-windows-amd64.exe $(CMD_PATH)

windows-386:
	mkdir -p bin
	CGO_ENABLED=0 GOOS=windows GOARCH=386 go build -ldflags="$(LDFLAGS)" -o bin/$(BINARY_NAME)-windows-386.exe $(CMD_PATH)

freebsd-amd64:
	mkdir -p bin
	CGO_ENABLED=0 GOOS=freebsd GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o bin/$(BINARY_NAME)-freebsd-amd64 $(CMD_PATH)

freebsd-386:
	mkdir -p bin
	CGO_ENABLED=0 GOOS=freebsd GOARCH=386 go build -ldflags="$(LDFLAGS)" -o bin/$(BINARY_NAME)-freebsd-386 $(CMD_PATH)

freebsd-arm64:
	mkdir -p bin
	CGO_ENABLED=0 GOOS=freebsd GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o bin/$(BINARY_NAME)-freebsd-arm64 $(CMD_PATH)

openbsd-amd64:
	mkdir -p bin
	CGO_ENABLED=0 GOOS=openbsd GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o bin/$(BINARY_NAME)-openbsd-amd64 $(CMD_PATH)

openbsd-386:
	mkdir -p bin
	CGO_ENABLED=0 GOOS=openbsd GOARCH=386 go build -ldflags="$(LDFLAGS)" -o bin/$(BINARY_NAME)-openbsd-386 $(CMD_PATH)

openbsd-arm64:
	mkdir -p bin
	CGO_ENABLED=0 GOOS=openbsd GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o bin/$(BINARY_NAME)-openbsd-arm64 $(CMD_PATH)

netbsd-amd64:
	mkdir -p bin
	CGO_ENABLED=0 GOOS=netbsd GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o bin/$(BINARY_NAME)-netbsd-amd64 $(CMD_PATH)

netbsd-386:
	mkdir -p bin
	CGO_ENABLED=0 GOOS=netbsd GOARCH=386 go build -ldflags="$(LDFLAGS)" -o bin/$(BINARY_NAME)-netbsd-386 $(CMD_PATH)

netbsd-arm64:
	mkdir -p bin
	CGO_ENABLED=0 GOOS=netbsd GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o bin/$(BINARY_NAME)-netbsd-arm64 $(CMD_PATH)

darwin-amd64:
	mkdir -p bin
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o bin/$(BINARY_NAME)-darwin-amd64 $(CMD_PATH)

darwin-arm64:
	mkdir -p bin
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o bin/$(BINARY_NAME)-darwin-arm64 $(CMD_PATH)

clean:
	rm -rf bin/

