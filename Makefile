.PHONY: help build build-all install test lint vet fmt clean run
.PHONY: build-linux build-linux-loong64 build-linux-musl build-darwin build-windows build-freebsd
.PHONY: dist dist-linux dist-darwin dist-windows dist-freebsd dist-tarball dist-zip dist-linux-loong64
.PHONY: clean-all checksums
.PHONY: npm-version npm-binaries npm-packages npm-pack npm-publish-all npm-publish-pre npm-publish

# Variables
BINARY_NAME=fd
VERSION=$(shell git describe --tags --abbrev=0 2>/dev/null || grep -oP 'const version = "\K[^"]+' cmd/fd/help.go 2>/dev/null || echo "10.4.2-go")
PRE_VERSION=$(if $(filter %-pre,$(VERSION)),$(VERSION),$(VERSION)-pre)
LDFLAGS=-ldflags "-s -w -X main.version=$(VERSION)"
GOBUILD_FLAGS=-trimpath
DIST_DIR=dist
CHECKSUM_FILE=$(DIST_DIR)/checksums.txt

# UPX compression (skip for macOS - not supported)
USE_UPX ?= true
ifeq ($(shell which upx 2>/dev/null),)
USE_UPX = false
endif
ifeq ($(USE_UPX),true)
UPX_CMD = upx -9
else
UPX_CMD = @true
endif

# Platforms and architectures (for reference)
# linux: amd64 arm64 loong64
# darwin: amd64 arm64
# windows: amd64 arm64

# Default target
help:
	@echo "go-fd Build System"
	@echo ""
	@echo "Build targets:"
	@echo "  build            Build for current platform"
	@echo "  build-linux      Build for Linux (amd64, arm64, arm, 386, loong64, riscv64, ppc64le, s390x)"
	@echo "  build-linux-loong64 Build for Linux LoongArch64"
	@echo "  build-linux-musl Build for Linux musl (amd64, arm64; static)"
	@echo "  build-darwin     Build for macOS (amd64, arm64)"
	@echo "  build-windows    Build for Windows (amd64, arm64, 386)"
	@echo "  build-freebsd    Build for FreeBSD (amd64, arm64)"
	@echo "  build-all        Build for all platforms and architectures"
	@echo ""
	@echo "Distribution targets:"
	@echo "  dist           Build all distribution packages (tar.gz + zip + checksums)"
	@echo "  dist-linux     Build Linux packages (tar.gz)"
	@echo "  dist-darwin    Build macOS packages (tar.gz)"
	@echo "  dist-freebsd   Build FreeBSD packages (tar.gz)"
	@echo "  dist-windows   Build Windows packages (zip)"
	@echo "  dist-linux-loong64 Build Linux LoongArch64 packages"
	@echo "  dist-tarball   Build tarball packages only"
	@echo "  dist-zip       Build zip packages only"
	@echo ""
	@echo "NPM targets:"
	@echo "  npm-version       Sync version to npm package"
	@echo "  npm-packages      Build platform-specific npm packages"
	@echo "  npm-pack          Pack main + all platform packages"
	@echo "  npm-publish-all   Publish main + all platform packages"
	@echo "  npm-publish-pre   Publish pre-release packages"
	@echo "  npm-binaries      [Legacy] Build all binaries into single package"
	@echo "  npm-publish       [Legacy] Publish main package only"
	@echo ""
	@echo "Other targets:"
	@echo "  install        Install via go install"
	@echo "  test           Run tests"
	@echo "  lint           Run linter (go vet)"
	@echo "  vet            Run go vet"
	@echo "  fmt            Format code"
	@echo "  clean          Remove build artifacts"
	@echo "  clean-all      Remove everything including dist"
	@echo "  checksums      Generate checksums for all dist files"
	@echo "  run            Build and run"
	@echo "  help           Show this help"

# Build for current platform
build:
	go build $(GOBUILD_FLAGS) $(LDFLAGS) -o bin/$(BINARY_NAME) ./cmd/fd

# Platform builds
# Linux covers the full practical architecture matrix.
build-linux:
	@echo "Building for Linux (amd64, arm64, arm, 386, loong64, riscv64, ppc64le, s390x)..."
	@mkdir -p bin
	GOOS=linux GOARCH=amd64 go build $(GOBUILD_FLAGS) $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-amd64 ./cmd/fd
	GOOS=linux GOARCH=arm64 go build $(GOBUILD_FLAGS) $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-arm64 ./cmd/fd
	GOOS=linux GOARCH=arm GOARM=7 go build $(GOBUILD_FLAGS) $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-arm ./cmd/fd
	GOOS=linux GOARCH=386 go build $(GOBUILD_FLAGS) $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-386 ./cmd/fd
	GOOS=linux GOARCH=loong64 go build $(GOBUILD_FLAGS) $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-loong64 ./cmd/fd
	GOOS=linux GOARCH=riscv64 go build $(GOBUILD_FLAGS) $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-riscv64 ./cmd/fd
	GOOS=linux GOARCH=ppc64le go build $(GOBUILD_FLAGS) $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-ppc64le ./cmd/fd
	GOOS=linux GOARCH=s390x go build $(GOBUILD_FLAGS) $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-s390x ./cmd/fd
	@echo "Compressing Linux amd64 binary with UPX..."
	$(UPX_CMD) bin/$(BINARY_NAME)-linux-amd64

build-linux-loong64:
	@echo "Building for Linux LoongArch64..."
	@mkdir -p bin
	GOOS=linux GOARCH=loong64 go build $(GOBUILD_FLAGS) $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-loong64 ./cmd/fd

# musl: static builds with CGO_ENABLED=0 (Alpine and other musl distros)
build-linux-musl:
	@echo "Building for Linux musl (amd64, arm64)..."
	@mkdir -p bin
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(GOBUILD_FLAGS) $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-musl-amd64 ./cmd/fd
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build $(GOBUILD_FLAGS) $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-musl-arm64 ./cmd/fd
	@echo "Compressing Linux musl amd64 binary with UPX..."
	$(UPX_CMD) bin/$(BINARY_NAME)-linux-musl-amd64

build-darwin:
	@echo "Building for macOS (amd64, arm64)..."
	@mkdir -p bin
	GOOS=darwin GOARCH=amd64 go build $(GOBUILD_FLAGS) $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-amd64 ./cmd/fd
	GOOS=darwin GOARCH=arm64 go build $(GOBUILD_FLAGS) $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-arm64 ./cmd/fd

build-windows:
	@echo "Building for Windows (amd64, arm64, 386)..."
	@mkdir -p bin
	GOOS=windows GOARCH=amd64 go build $(GOBUILD_FLAGS) $(LDFLAGS) -o bin/$(BINARY_NAME)-windows-amd64.exe ./cmd/fd
	GOOS=windows GOARCH=arm64 go build $(GOBUILD_FLAGS) $(LDFLAGS) -o bin/$(BINARY_NAME)-windows-arm64.exe ./cmd/fd
	GOOS=windows GOARCH=386 go build $(GOBUILD_FLAGS) $(LDFLAGS) -o bin/$(BINARY_NAME)-windows-386.exe ./cmd/fd
	@echo "Compressing Windows amd64 binary with UPX..."
	$(UPX_CMD) bin/$(BINARY_NAME)-windows-amd64.exe

build-freebsd:
	@echo "Building for FreeBSD (amd64, arm64)..."
	@mkdir -p bin
	GOOS=freebsd GOARCH=amd64 go build $(GOBUILD_FLAGS) $(LDFLAGS) -o bin/$(BINARY_NAME)-freebsd-amd64 ./cmd/fd
	GOOS=freebsd GOARCH=arm64 go build $(GOBUILD_FLAGS) $(LDFLAGS) -o bin/$(BINARY_NAME)-freebsd-arm64 ./cmd/fd

# Build all platforms
build-all: build-linux build-linux-musl build-darwin build-windows build-freebsd
	@echo ""
	@echo "Build complete! Binaries in bin/"
	@ls -lh bin/

# Install
install:
	go install $(GOBUILD_FLAGS) $(LDFLAGS) ./cmd/fd

# Test
test:
	go test -race ./...

# Lint
lint:
	go vet ./...

# Vet
vet: lint

# Format
fmt:
	gofmt -w .
	goimports -w . 2>/dev/null || go fmt ./...

# Distribution: tarballs (Linux + macOS + FreeBSD)
dist-tarball: build-linux build-linux-musl build-darwin build-freebsd
	@echo ""
	@echo "Creating tarball packages..."
	@for arch in amd64 arm64 arm 386 loong64 riscv64 ppc64le s390x; do \
		echo "  Packaging $(BINARY_NAME)-linux-$${arch}.tar.gz..."; \
		./scripts/build-tarball.sh linux $${arch} $(VERSION); \
	done
	@for arch in amd64 arm64; do \
		echo "  Packaging $(BINARY_NAME)-darwin-$${arch}.tar.gz..."; \
		./scripts/build-tarball.sh darwin $${arch} $(VERSION); \
	done
	@for arch in amd64 arm64; do \
		echo "  Packaging $(BINARY_NAME)-freebsd-$${arch}.tar.gz..."; \
		./scripts/build-tarball.sh freebsd $${arch} $(VERSION); \
	done
	@for arch in amd64 arm64; do \
		echo "  Packaging $(BINARY_NAME)-linux-musl-$${arch}.tar.gz..."; \
		./scripts/build-tarball.sh linux-musl $${arch} $(VERSION); \
	done

dist-zip: build-windows
	@echo ""
	@echo "Creating Windows zip packages..."
	@for arch in amd64 arm64 386; do \
		echo "  Packaging $(BINARY_NAME)-windows-$${arch}.zip..."; \
		./scripts/build-zip.sh $${arch} $(VERSION); \
	done

dist-linux: dist-tarball
	@echo "Linux packages built."

dist-linux-loong64: build-linux-loong64
	@echo ""
	@echo "Creating LoongArch64 tarball..."
	./scripts/build-tarball.sh linux loong64 $(VERSION)

dist-darwin: dist-tarball
	@echo "macOS packages built."

dist-freebsd: dist-tarball
	@echo "FreeBSD packages built."

dist-windows: dist-zip
	@echo "Windows packages built."

checksums:
	@echo "Generating checksums..."
	@cd $(DIST_DIR) && \
	find . -type f \( -name "*.tar.gz" -o -name "*.zip" \) | sort | \
	while read f; do \
		sha256sum "$$f"; \
	done > checksums.txt
	@echo "Checksums written to $(CHECKSUM_FILE)"
	@cat $(CHECKSUM_FILE)

dist: dist-tarball dist-zip checksums
	@echo ""
	@echo "=========================================="
	@echo "All distribution packages built!"
	@echo ""
	@echo "Location: $(DIST_DIR)/"
	@echo ""
	@ls -lh $(DIST_DIR)/*/* 2>/dev/null || true
	@echo ""
	@echo "Checksums: $(CHECKSUM_FILE)"
	@echo "=========================================="

# NPM targets
npm-version:
	./scripts/sync-npm-version.sh $(VERSION)

# Legacy: build all binaries into single package (use npm-packages instead)
npm-binaries: build-all
	@echo "WARNING: npm-binaries is deprecated, use npm-packages instead" >&2
	./scripts/build-npm.sh

# Build platform-specific npm packages (optionalDependencies architecture)
npm-packages: build-all
	./scripts/build-npm-packages.sh

npm-pack: npm-version npm-packages
	@echo "Packing platform packages..."
	@for d in npm/packages/*/; do \
		if [ -f "$$d/package.json" ]; then \
			echo "  Packing $$(basename $$d)..."; \
			cd "$$d" && npm pack && cd - > /dev/null; \
			mv "$$d"/*.tgz npm/ 2>/dev/null || true; \
		fi; \
	done
	@echo "Packing main package..."
	cd npm && npm pack
	@echo "Done. Tarballs in npm/"

npm-publish-all: npm-version npm-packages
	@echo "Publishing platform packages..."
	@for d in npm/packages/*/; do \
		if [ -f "$$d/package.json" ]; then \
			echo "  Publishing $$(basename $$d)..."; \
			cd "$$d" && npm publish --tag latest && cd - > /dev/null; \
		fi; \
	done
	@echo "Publishing main package..."
	cd npm && npm publish --tag latest
	@echo "Published all packages!"

npm-publish-pre:
	./scripts/sync-npm-version.sh $(PRE_VERSION)
	$(MAKE) npm-packages VERSION=$(PRE_VERSION)
	@echo "Publishing pre-release platform packages..."
	@for d in npm/packages/*/; do \
		if [ -f "$$d/package.json" ]; then \
			echo "  Publishing $$(basename $$d)..."; \
			cd "$$d" && npm publish --tag next && cd - > /dev/null; \
		fi; \
	done
	@echo "Publishing main package (next)..."
	cd npm && npm publish --tag next

# Legacy: publish main package only (use npm-publish-all instead)
npm-publish: npm-version npm-binaries
	@echo "WARNING: npm-publish is deprecated, use npm-publish-all instead" >&2
	cd npm && npm publish --tag latest

# Clean
clean:
	rm -rf bin/
	rm -rf npm/packages/
	rm -rf npm/bin/
	rm -f npm/*.tgz

# Clean all
clean-all: clean
	rm -rf $(DIST_DIR)

# Run
run: build
	./bin/$(BINARY_NAME)
