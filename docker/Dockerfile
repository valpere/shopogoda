FROM golang:1.24.6-alpine AS builder

# Install git and ca-certificates
RUN apk update && apk upgrade && apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o shopogoda cmd/bot/main.go

# Final stage
FROM alpine:latest

# Install ca-certificates and timezone data
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN adduser -D -s /bin/sh shopogoda

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/shopogoda .

# Create configs directory (COPY can't use shell operators)
RUN mkdir -p ./configs

# Change ownership to non-root user
RUN chown -R shopogoda:shopogoda /app

# Switch to non-root user
USER shopogoda

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the application
CMD ["./shopogoda"]
