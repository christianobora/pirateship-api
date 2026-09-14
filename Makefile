.PHONY: check fmt test vet lint vuln

check: fmt vet test lint vuln

fmt:
	test -z "$$(gofmt -l .)"

test:
	go test -race -coverprofile=coverage.out ./...

vet:
	go vet ./...

lint:
	golangci-lint run ./...

vuln:
	govulncheck ./...
