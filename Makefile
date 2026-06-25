.PHONY: fmt vet test build smoke real-opencode-e2e check clean

fmt:
	gofmt -w .

test:
	go test ./...

vet:
	go vet ./...

build:
	go build -o agentctl .

smoke: build
	scripts/smoke.sh

real-opencode-e2e: build
	scripts/real-opencode-e2e.sh

check: fmt vet test build smoke

clean:
	rm -f agentctl
