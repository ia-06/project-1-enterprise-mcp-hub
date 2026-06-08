FROM golang:alpine AS builder

WORKDIR /app

# Install dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
# We use cmd/server as the entrypoint per the project structure
RUN go build -o server ./cmd/server

# Multi-stage build for a tiny footprint
FROM alpine:latest

WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/server .

# Create a non-root user for security
RUN adduser -D mcpuser
USER mcpuser

EXPOSE 8080

CMD ["./server"]
