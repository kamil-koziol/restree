.PHONY: lint
lint:
	golangci-lint run

.PHONY: lint-fix
lint-fix:
	golangci-lint run

.PHONY: format
format:
	golangci-lint fmt

.PHONY: format-check
format-check:
	golangci-lint fmt -d
