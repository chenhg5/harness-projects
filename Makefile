APP         := hp
CMD         := ./cmd/hp
DIST        := dist
VERSION     ?= v0.0.10
COMMIT      := $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
BUILD_TIME  := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')

LDFLAGS := -s -w \
  -X github.com/chenhg5/harness-projects/internal/cli.Version=$(VERSION) \
  -X main.commit=$(COMMIT) \
  -X main.buildTime=$(BUILD_TIME)

PLATFORMS := darwin/arm64 darwin/amd64 linux/amd64 linux/arm64 windows/amd64 windows/arm64
SHA256 := $(shell command -v sha256sum >/dev/null 2>&1 && echo sha256sum || echo "shasum -a 256")

.PHONY: build test vet check release-all clean

build:
	go build -ldflags "$(LDFLAGS)" -o bin/$(APP) $(CMD)

test:
	go test -count=1 ./...

vet:
	test -z "$$(gofmt -l .)" || (echo "gofmt needed"; exit 1)
	go vet ./...

check: vet test

release-all: clean check
	@mkdir -p $(DIST)
	@$(foreach platform,$(PLATFORMS), \
		$(eval GOOS := $(word 1,$(subst /, ,$(platform)))) \
		$(eval GOARCH := $(word 2,$(subst /, ,$(platform)))) \
		$(eval EXT := $(if $(filter windows,$(GOOS)),.exe,)) \
		$(eval NAME := $(APP)-$(VERSION)-$(GOOS)-$(GOARCH)) \
		echo "Building $(NAME)" && mkdir -p $(DIST)/$(NAME) && \
		GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(DIST)/$(NAME)/$(APP)$(EXT) $(CMD) && \
		cp README.md README.zh-CN.md INSTALL.md LICENSE $(DIST)/$(NAME)/ && \
		$(if $(EXT),(cd $(DIST) && zip -qr $(NAME).zip $(NAME)),tar -C $(DIST) -czf $(DIST)/$(NAME).tar.gz $(NAME)) && \
		rm -rf $(DIST)/$(NAME);) true
	@cd $(DIST) && $(SHA256) *.tar.gz *.zip > checksums.txt
	@echo "Done: $(DIST)/"; ls -1 $(DIST)

clean:
	rm -rf $(DIST) bin/$(APP)
