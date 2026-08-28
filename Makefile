.PHONY: build test check clean

build:
	go build -o ./wenmar ./cmd/wenmar

test:
	go test ./... -v

check: build test
	@echo "Checking SDK drift..."
	@(cd ../wenmar-sdk/go && go test ./...)
	./wenmar --help > /dev/null
	./wenmar commands --json | jq . > /dev/null

clean:
	rm -f wenmar
