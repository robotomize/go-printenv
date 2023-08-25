.PHONY: test
test:
	go test -race -v ./...

.PHONY: test
test-cover:
	@go test -race -v -tags=all -cover ./... -coverprofile=coverage.out

.PHONY: lint
lint:
	golangci-lint run --timeout 5m -v ./...