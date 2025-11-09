.PHONY: help dev build-frontend build-backend build clean install-frontend install-backend docker

help:
	@echo "LaTeX Editor - Makefile"
	@echo ""
	@echo "Available targets:"
	@echo "  install-frontend  - Install frontend dependencies"
	@echo "  install-backend   - Install backend dependencies"
	@echo "  dev               - Run development servers"
	@echo "  build-frontend    - Build frontend for production"
	@echo "  build-backend     - Build backend binary"
	@echo "  build             - Build complete application"
	@echo "  clean             - Clean build artifacts"
	@echo "  docker            - Build Docker image"

install-frontend:
	cd frontend && npm install

install-backend:
	go mod download

dev:
	@echo "Starting development servers..."
	@echo "Backend will run on :8080"
	@echo "Frontend will run on :5173"
	cd frontend && npm run dev &
	go run cmd/server/main.go --db="${DATABASE_URL}" --state-dir=./data

build-frontend:
	cd frontend && npm run build

build-backend: build-frontend
	go build -o server cmd/server/main.go

build: build-frontend build-backend
	@echo "Build complete! Binary: ./server"

clean:
	rm -f server
	rm -rf web/dist/*
	rm -rf frontend/dist
	rm -rf frontend/node_modules
	rm -rf data

docker: build
	docker build -t latex-editor .
