VER ?= $(VERSION)
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

# LDFLAGS: use double quotes around values to preserve spaces/timestamps
LDFLAGS := -X "main.BuildTimestamp=$(shell date -u '+%Y-%m-%d %H:%M:%S')" \
	-X "main.VERSION=$(VER)" \
	-X "main.goVersion=$(shell go version | sed -r 's/go version go(.*)\ .*/\1/')"

# GO wrapper to ensure modules on and CGO off for static binaries
GO := GO111MODULE=on CGO_ENABLED=0 go
GOLANGCI_LINT_VERSION = v1.61.0

APP_NAME := BPB-Wizard
OUT_DIR := bin
DIST_DIR := dist

.PHONY: build clean

build:
	@set -e; \
	mkdir -p "$(OUT_DIR)" "$(DIST_DIR)"; \
	if [ "$(GOOS)" = "windows" ]; then \
		ext=".exe"; \
	else \
		ext=""; \
	fi; \
	echo "Building for $(GOOS)-$(GOARCH)..."; \
	outdir="$(OUT_DIR)/$(APP_NAME)-$(GOOS)-$(GOARCH)"; \
	mkdir -p "$$outdir"; \
	echo "  output dir: $$outdir"; \
	echo "  running: GOOS=$(GOOS) GOARCH=$(GOARCH) $(GO) build -trimpath -ldflags '$(LDFLAGS)' -o \"$$outdir/$(APP_NAME)$$ext\""; \
	GOOS=$(GOOS) GOARCH=$(GOARCH) $(GO) build -trimpath -ldflags '$(LDFLAGS)' -o "$$outdir/$(APP_NAME)$$ext"; \
	# copy LICENSE if exists (don't fail if missing) \
	if [ -f LICENSE ]; then cp LICENSE "$$outdir/"; fi; \
	archive="$(DIST_DIR)/$(APP_NAME)-$(GOOS)-$(GOARCH)"; \
	if [ -d "$$outdir" ]; then \
		if [ "$(GOOS)" = "windows" ] || [ "$(GOOS)" = "darwin" ]; then \
			echo "  creating zip: $$archive.zip"; \
			zip -j -q "$$archive.zip" "$$outdir"/*; \
		else \
			echo "  creating tar.gz: $$archive.tar.gz"; \
			tar -czf "$$archive.tar.gz" -C "$$outdir" .; \
		fi; \
	else \
		echo "Error: expected output directory $$outdir does not exist"; \
		exit 1; \
	fi

clean:
	@rm -rf "$(OUT_DIR)" "$(DIST_DIR)"
