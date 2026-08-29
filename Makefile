.PHONY: build test check check-published clean generate test-generated

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

# Test the generated commands (excludes hand-written ones via build tags).
test-generated: generate
	go build -tags generated ./...
	go test -tags generated ./cmd/... -run "TestVendors|TestLocations|TestAccount|TestDriversShow|TestDriversDelete|TestVehiclesDelete|TestVehiclesShow|TestStatements|TestCustomersShow" -v

clean:
	rm -f wenmar
