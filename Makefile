SHELL := /bin/sh
GO ?= go
VERSION ?= dev
DIST ?= dist
export VERSION DIST

TARGETS := darwin_amd64 darwin_arm64 linux_386 linux_amd64 linux_armv7 linux_arm64

.PHONY: build test release

build:
	@mkdir -p bin
	$(GO) build -trimpath -buildvcs=false -ldflags "-s -w -X github.com/dacalin/aifactory/internal/commands.Version=$${VERSION#v}" -o "bin/aifactory" .

test:
	$(GO) test ./...
	$(GO) -C .ai-factory/rules test ./...

# Release builds need Make, Go, tar and sha256sum or shasum.
release:
	@set -eu; \
	case "$$VERSION" in dev|""|.|..|*[!A-Za-z0-9._-]*) echo 'Use make release VERSION=vX.Y.Z' >&2; exit 1;; esac; \
	command -v tar >/dev/null; \
	if command -v sha256sum >/dev/null; then checksum='sha256sum'; else command -v shasum >/dev/null; checksum='shasum -a 256'; fi; \
	mkdir -p "$$DIST/$$VERSION"; \
	out=$$(cd "$$DIST/$$VERSION" && pwd); \
	for target in $(TARGETS); do \
		os=$${target%_*}; arch=$${target#*_}; \
		if [ "$$arch" = armv7 ]; then arch=arm; fi; \
		stage="$$out/$$target"; mkdir -p "$$stage"; \
		CGO_ENABLED=0 GOOS="$$os" GOARCH="$$arch" GOARM=7 GOAMD64=v1 \
			$(GO) build -trimpath -buildvcs=false -ldflags "-s -w -X github.com/dacalin/aifactory/internal/commands.Version=$${VERSION#v}" -o "$$stage/aifactory" .; \
		cp README.md "$$stage/README.md"; \
		archive="$$out/aifactory_$${VERSION}_$$target"; \
		COPYFILE_DISABLE=1 tar -czf "$$archive.tar.gz" -C "$$stage" aifactory README.md; \
		printf 'Built %s\n' "$$target"; \
	done; \
	(cd "$$out" && $$checksum aifactory_"$$VERSION"_*.tar.gz > SHA256SUMS); \
	printf 'Release archives and checksums: %s\n' "$$out"
