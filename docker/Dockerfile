FROM golang:1.26-alpine AS builder

# Install git and ca-certificates
RUN apk update && apk upgrade && apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build arguments for version information
ARG VERSION=dev
ARG GIT_COMMIT=unknown
ARG BUILD_TIME=unknown

# Build the application with version information
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo \
    -ldflags "-X github.com/valpere/shopogoda/internal/version.Version=${VERSION} \
              -X github.com/valpere/shopogoda/internal/version.GitCommit=${GIT_COMMIT} \
              -X github.com/valpere/shopogoda/internal/version.BuildTime=${BUILD_TIME}" \
    -o shopogoda cmd/bot/main.go

# Final stage
FROM alpine:latest

# Install ca-certificates and timezone data
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
# Fixed uid/gid so volume ownership (k8s fsGroup, bind mounts) is predictable
RUN addgroup -g 10001 shopogoda && adduser -D -u 10001 -G shopogoda -s /bin/sh shopogoda

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/shopogoda .

# Create configs directory (COPY can't use shell operators) and the SQLite data directory
RUN mkdir -p ./configs ./data

# Change ownership to non-root user
RUN chown -R shopogoda:shopogoda /app

# Switch to non-root user
USER shopogoda

# SQLite database lives here; mount a volume to persist it
ENV DB_PATH=/app/data/shopogoda.db
VOLUME /app/data

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the application
CMD ["./shopogoda"]
