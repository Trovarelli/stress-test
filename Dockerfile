# Stage 1: Build binary
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install ca-certificates to allow HTTPS requests
RUN apk add --no-cache ca-certificates tzdata

# Cache dependencies
COPY go.mod ./
RUN go mod download

# Copy source code
COPY . .

# Build static binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /stresstest ./cmd/stresstest

# Stage 2: Final lightweight image
FROM scratch

# Copy CA certificates for HTTPS support
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
# Copy timezone info if needed
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy compiled binary
COPY --from=builder /stresstest /stresstest

# Set entrypoint so CLI parameters can be supplied directly to 'docker run'
ENTRYPOINT ["/stresstest"]
