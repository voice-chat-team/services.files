# Build stage
FROM golang:1.27-alpine AS builder

WORKDIR /app

# Install dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY cmd ./cmd/
COPY internal ./internal/

# Build a static binary
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /app/files ./cmd/files

# Production stage
FROM alpine:3.22 AS runtime

# CA certificates for TLS to S3
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/files ./files

EXPOSE 5060

CMD ["./files"]
