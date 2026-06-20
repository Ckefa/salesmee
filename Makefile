.PHONY: build run dev clean test deploy css css-watch install

# Build CSS
css:
	npm run css:build

# Watch CSS
css-watch:
	npm run css:watch

# Build the application (with CSS)
build: css
	go build -o bin/app cmd/server/main.go

# Run the application
run: build
	./bin/app

# Development with hot reload
dev:
	npm run css:build
	node scripts/build-js.js
	air

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f internal/db/app.db

# Run tests
test:
	go test ./... -count=1
	go vet ./...

# Production build
build-prod: css
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o bin/app-linux cmd/server/main.go
	GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o bin/app-darwin cmd/server/main.go
	GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o bin/app-windows.exe cmd/server/main.go

# Deploy (example for Docker)
deploy:
	docker build -t  .
	docker run -p 8080:8080 

# Install dependencies (Go + Node)
install:
	go mod download
	go mod tidy
	npm install
