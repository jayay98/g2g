CMDS ?= git-g2g git-remote-g2g
# https://pkg.go.dev/go/build
VERSION := $(shell git describe --tags --always --dirty)

.PHONY: all
all: fmt unit build

.PHONY: clean
clean:
	$(foreach obj, $(CMDS), go clean -C cmd/$(obj) -i;)

.PHONY: install
install:
	$(foreach obj, $(CMDS), go install -C cmd/$(obj);)

.PHONY: build
build:
	$(foreach obj, $(CMDS), go build -C cmd/$(obj) || exit 1;)

.PHONY: fmt
fmt:
	gofmt -w -l .

# Unit tests
.PHONY: test
test:
	go test g2g/pkg/pack -count=1

.PHONY: unit
e2e: install
	go test g2g/tests -count=1