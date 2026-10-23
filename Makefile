.PHONY: test test-integration

TEST_DB ?= postgres://spendwise:spendwise@localhost:5433/postgres?sslmode=disable

test:
	go test -race ./...

test-integration:
	TEST_DATABASE_URL="$(TEST_DB)" go test ./tests/... -count=1