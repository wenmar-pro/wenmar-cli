.PHONY: build test check check-published clean generate surface-snapshot surface-diff golden-update

build:
	go build -o ./wenmar ./cmd/wenmar

test:
	go test ./... -v

check: build test
	@echo "Checking SDK drift..."
	@(cd ../wenmar-sdk/go && go test ./...)
	./wenmar --help > /dev/null
	./wenmar commands --json | jq . > /dev/null

# Verify the CLI builds/tests against the published SDK tag, bypassing the
# local go.work workspace. Catches broken published tags before they hit CI.
check-published:
	GOWORK=off go build -o /dev/null ./cmd/wenmar
	GOWORK=off go test ./...

# Generate CLI commands from the enriched OpenAPI spec.
# Requires the wenmar-sdk worktree at ../wenmar-sdk (or override SPEC_PATH).
SPEC_PATH ?= ../wenmar-sdk/spec/openapi.enriched.yaml
generate:
	go run ./cmd/gencli -spec $(SPEC_PATH) -overrides cmd/gen_overrides.yaml -out cmd/

# Refresh the committed golden fixtures (regen-drift gate).
golden-update:
	go run ./cmd/gencli -spec $(SPEC_PATH) -overrides cmd/gen_overrides.yaml -build-tag ignore -out cmd/golden

clean:
	rm -f wenmar

# Dump the command surface as JSON for CI diffing.
surface-snapshot:
	go run ./cmd/wenmar surface-snapshot > surface-snapshot.json
	@echo "Snapshot written to surface-snapshot.json"

# Diff the current surface against the stored snapshot.
# Fails if the command tree changed (catches breaking changes in CI).
surface-diff:
	go run ./cmd/wenmar surface-snapshot > /tmp/surface-current.json
	@if [ -f surface-snapshot.json ]; then \
		diff -u surface-snapshot.json /tmp/surface-current.json && echo "No surface changes." || (echo "Surface changed! Update with: make surface-snapshot" && exit 1); \
	else \
		echo "No stored snapshot. Run: make surface-snapshot"; \
	fi
