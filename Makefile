.PHONY: fmt vet test build check clean

fmt:
	gofmt -w .

test:
	go test ./...

vet:
	go vet ./...

build:
	go build -o agentctl .

check: fmt vet test build

clean:
	rm -f agentctl
