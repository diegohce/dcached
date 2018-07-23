GOOS=linux
GOFILES=dcached.go config.go cache.go siblings.go request_handlers.go request_stats.go request_import.go
VERSION=0.1.0
BINARY_NAME=dcached

LDFLAGS=-ldflags "-X main.VERSION=$(VERSION)"

ifeq ($(GOOS), windows)
	EXE=.exe
endif

all: deps build

build:
	@GOOS=$(GOOS) go build -o $(BINARY_NAME)$(EXE) $(LDFLAGS) $(GOFILES)

run:
	@GOOS=$(GOOS) go run $(LDFLAGS) $(GOFILES)

.PHONY clean:
	@rm -f $(BINARY_NAME) $(BINARY_NAME).exe

deps:
	@go get github.com/julienschmidt/httprouter
	@go get github.com/creamdog/gonfig
