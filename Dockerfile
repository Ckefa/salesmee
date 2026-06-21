FROM golang:1.26-alpine AS builder
RUN apk add --no-cache nodejs npm
WORKDIR /app

COPY package.json package-lock.json ./
RUN npm ci

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN mkdir -p web/static/uploads/logos web/static/uploads/products
RUN npm run css:build
RUN npm run css:build-landing
RUN npm run js:build
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/app ./cmd/server/main.go

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/app .
COPY --from=builder /app/web ./web
EXPOSE 8080
CMD ["./app"]
