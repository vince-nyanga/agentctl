.PHONY: fmt test build check clean

fmt:
	gofmt -w .

test:
	go test ./...

build:
	go build -o agentctl .

check: fmt test build

clean:
	rm -f agentctl
