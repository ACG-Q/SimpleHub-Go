.PHONY: build build-frontend build-release run clean dev dev-backend dev-frontend

VERSION ?= $(shell date +%Y%m%d-%H%M%S)
LDFLAGS := -s -w -X main.version=$(VERSION)
GOOS := $(shell go env GOOS)
BIN := server$(if $(filter windows,$(GOOS)),.exe)

# --- Development ---

dev-backend:
	CGO_ENABLED=0 go run ./cmd/server

dev-frontend:
	cd web && npm run dev

# --- Build ---

build-frontend:
	cd web && npm ci && npm run build

build: build-frontend
	cp -r web/dist cmd/server/dist
	CGO_ENABLED=0 go build -o bin/$(BIN) -ldflags="$(LDFLAGS)" ./cmd/server

# Quick build (frontend already built)
build-quick: build-frontend
	cp -r web/dist cmd/server/dist
	CGO_ENABLED=0 go build -o bin/$(BIN) -ldflags="$(LDFLAGS)" ./cmd/server

# Go-only build (uses placeholder frontend)
build-go:
	CGO_ENABLED=0 go build -o bin/$(BIN) ./cmd/server

# --- Release ---

build-release:
	./scripts/build-release.sh

# --- Run ---

run:
	./bin/$(BIN)

# --- Clean ---

clean:
	rm -rf bin/ releases/

# --- Docker ---

docker-build:
	docker build -t simplehub-go:$(VERSION) .
	docker tag simplehub-go:$(VERSION) simplehub-go:latest

docker-run:
	docker run -d --name simplehub-go -p 3000:3000 \
		-v simplehub-data:/app/data \
		simplehub-go:latest
