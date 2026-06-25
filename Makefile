.PHONY: fmt vet test build smoke check clean

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

check: fmt vet test build smoke

clean:
	rm -f agentctl
