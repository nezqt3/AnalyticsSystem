SHELL := /bin/zsh
.DEFAULT_GOAL := help

BACKEND_DIR := backend
FRONTEND_DIR := frontend
GO_RUN := cd $(BACKEND_DIR) && go run ./cmd/server
NPM := cd $(FRONTEND_DIR) && npm

.PHONY: help install install-frontend install-pre-commit dev dev-backend dev-frontend build build-backend build-frontend lint lint-backend lint-frontend format typecheck test test-backend test-frontend test-coverage test-coverage-backend test-coverage-frontend ci pre-commit-install docker-up docker-down clean

help:
	@printf "Available commands:\n"
	@printf "  make install             Install frontend dependencies\n"
	@printf "  make install-pre-commit  Install git hooks via pre-commit\n"
	@printf "  make dev-backend         Run Go backend locally\n"
	@printf "  make dev-frontend        Run Vite frontend locally\n"
	@printf "  make build              Build backend and frontend\n"
	@printf "  make lint               Run linters\n"
	@printf "  make format             Format frontend and Go code\n"
	@printf "  make typecheck          Run frontend type checks\n"
	@printf "  make test               Run available tests\n"
	@printf "  make test-coverage      Generate test coverage for backend and frontend\n"
	@printf "  make ci                 Full local CI pipeline\n"
	@printf "  make docker-up          Start docker-compose stack\n"
	@printf "  make docker-down        Stop docker-compose stack\n"

install: install-frontend

install-frontend:
	$(NPM) install

install-pre-commit:
	pre-commit install

dev:
	@printf "Run in separate terminals:\n"
	@printf "  make dev-backend\n"
	@printf "  make dev-frontend\n"

dev-backend:
	$(GO_RUN)

dev-frontend:
	$(NPM) run dev

build: build-backend build-frontend

build-backend:
	cd $(BACKEND_DIR) && go build ./cmd/server

build-frontend:
	$(NPM) run build

lint: lint-backend lint-frontend

lint-backend:
	cd $(BACKEND_DIR) && test -z "$$(gofmt -l $$(find . -name '*.go' -not -path './data/*'))"
	cd $(BACKEND_DIR) && go vet ./...

lint-frontend:
	$(NPM) run lint

format:
	cd $(BACKEND_DIR) && gofmt -w $$(find . -name '*.go' -not -path './data/*')
	$(NPM) run format

typecheck:
	$(NPM) run typecheck

test:
	$(MAKE) test-backend
	$(MAKE) test-frontend

test-backend:
	cd $(BACKEND_DIR) && go test ./...

test-frontend:
	$(NPM) run test

test-coverage:
	$(MAKE) test-coverage-backend
	$(MAKE) test-coverage-frontend

test-coverage-backend:
	cd $(BACKEND_DIR) && go test ./... -coverprofile=coverage.out

test-coverage-frontend:
	$(NPM) run test:coverage

ci: lint typecheck test build

docker-up:
	docker compose up --build

docker-down:
	docker compose down

clean:
	cd $(BACKEND_DIR) && go clean
	cd $(FRONTEND_DIR) && rm -rf dist
