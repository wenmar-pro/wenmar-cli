.PHONY: build test check check-published clean

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

clean:
	rm -f wenmar
