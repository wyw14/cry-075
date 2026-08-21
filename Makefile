.PHONY: dev test race vet migrate seed web-test web-build

dev:
	go run ./cmd/server

test:
	go test ./...

race:
	go test -race ./...

vet:
	go vet ./...

migrate:
	psql "$(DATABASE_URL)" -v ON_ERROR_STOP=1 -f migrations/001_initial.sql -f migrations/002_indexes.sql

seed:
	psql "$(DATABASE_URL)" -v ON_ERROR_STOP=1 -f migrations/003_seed.sql

web-test:
	npm --prefix web test

web-build:
	npm --prefix web run build

