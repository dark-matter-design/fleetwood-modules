SHELL := /bin/bash

# The Go toolchain and the fleetwood-mod CLI both live in the Fleetwood checkout, which is private
# and not fetchable with "go run". Point this at it if it is not beside this repository.
FLEETWOOD ?= ../fleetwood
GOENV := source $(FLEETWOOD)/.tools/env.sh 2>/dev/null || true;

# Baked into index.json at build time, so it has to match where the repository is actually served.
BASE ?= https://dark-matter-design.github.io/fleetwood-modules
REPO_NAME ?= Fleetwood Modules
PUBLISHER ?= Dark Matter Design

# Published output. GitHub Pages serves this directory from the default branch, so it is committed.
OUT := docs
BUILD := build
KEYS := .keys
MOD := $(BUILD)/fleetwood-mod

.PHONY: all keygen tools packages publish verify clean

all: publish

# Creates the repository signing key once. The private key stays in .keys/, which is gitignored:
# it is a development key for the demo repository and never leaves this machine.
keygen: tools
	@mkdir -p $(KEYS)
	@test -f $(KEYS)/modsign.key || $(MOD) keygen -out $(KEYS) -name modsign
	@echo "Public key:"
	@cat $(KEYS)/modsign.pub

tools:
	@test -d $(FLEETWOOD) || { echo "Fleetwood checkout not found at $(FLEETWOOD); set FLEETWOOD=/path/to/fleetwood" >&2; exit 1; }
	@mkdir -p $(BUILD)
	@$(GOENV) cd $(FLEETWOOD) && go build -o "$(CURDIR)/$(MOD)" ./cmd/fleetwood-mod

# Builds every module under modules/ to wasip1 WebAssembly and packs each into a .fwmod. The
# package filename is <directory>-<manifest version>.fwmod, which is what the index refers to.
packages: tools
	@rm -rf $(BUILD)/packages && mkdir -p $(BUILD)/packages
	@set -e; for dir in modules/*/; do \
		name=$$(basename "$$dir"); \
		version=$$(sed -n 's/^version:[[:space:]]*//p' "$$dir/manifest.yaml" | head -1 | tr -d '"'); \
		if [ -z "$$version" ]; then echo "no version in $$dir/manifest.yaml" >&2; exit 1; fi; \
		echo "building $$name $$version"; \
		$(GOENV) sh -c "cd '$$dir' && GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o module.wasm ."; \
		$(MOD) pack -in "$$dir" -out "$(BUILD)/packages/$$name-$$version.fwmod"; \
		rm -f "$$dir/module.wasm"; \
	done

# Signs the packages and writes the repository (index.json, its signature, and each package with
# its own signature) into docs/ for Pages to serve.
publish: packages
	@test -f $(KEYS)/modsign.key || { echo "no signing key; run 'make keygen' first" >&2; exit 1; }
	@rm -rf $(OUT) && mkdir -p $(OUT)
	@touch $(OUT)/.nojekyll
	$(MOD) repo -in $(BUILD)/packages -out $(OUT) -key $(KEYS)/modsign.key \
		-base "$(BASE)" -name "$(REPO_NAME)" -publisher "$(PUBLISHER)"
	@cp $(KEYS)/modsign.pub $(OUT)/minisign.pub
	@chmod 644 $(OUT)/* $(OUT)/.nojekyll
	@echo
	@echo "Repository URL: $(BASE)"
	@echo "Public key:"
	@cat $(KEYS)/modsign.pub

# Proves the committed repository is internally consistent: every package the index names exists
# and passes the checks core runs, its SHA-256 matches, and its URL sits under the published base.
# Signatures are checked too when the minisign CLI is installed; core verifies them regardless.
verify: tools
	@set -e; test -f $(OUT)/index.json || { echo "no $(OUT)/index.json; run 'make publish'" >&2; exit 1; }
	@set -e; jq -e '.schema == 1 and (.modules | length) > 0' $(OUT)/index.json >/dev/null \
		|| { echo "$(OUT)/index.json is not a usable index" >&2; exit 1; }
	@set -e; jq -r '.modules[].versions[] | [.url, .sha256] | @tsv' $(OUT)/index.json | \
	while IFS=$$'\t' read -r url want; do \
		case "$$url" in "$(BASE)/"*) ;; *) echo "url $$url is not under $(BASE)" >&2; exit 1;; esac; \
		file="$(OUT)/$${url##*/}"; \
		test -f "$$file" || { echo "index names $$file, which is missing" >&2; exit 1; }; \
		test -f "$$file.minisig" || { echo "$$file has no signature" >&2; exit 1; }; \
		got=$$(shasum -a 256 "$$file" | cut -d' ' -f1); \
		test "$$got" = "$$want" || { echo "$$file sha256 $$got does not match the index ($$want)" >&2; exit 1; }; \
		$(MOD) validate -in "$$file" >/dev/null || exit 1; \
		if command -v minisign >/dev/null; then minisign -Vqm "$$file" -p $(KEYS)/modsign.pub >/dev/null || exit 1; fi; \
		echo "ok $$file"; \
	done
	@if command -v minisign >/dev/null; then minisign -Vqm $(OUT)/index.json -p $(KEYS)/modsign.pub >/dev/null && echo "ok $(OUT)/index.json signature"; \
	else echo "note: minisign is not installed, so signatures were not checked here"; fi

clean:
	rm -rf $(BUILD)
