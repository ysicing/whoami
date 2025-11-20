# Build stage
FROM ysicing/god AS builder

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum* ./
RUN go mod download || true

# Copy source code
COPY . .

# Build arguments for version info
ARG VERSION=dev
ARG GIT_COMMIT=unknown
ARG BUILD_TIME=unknown

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s \
    -X 'main.Version=${VERSION}' \
    -X 'main.GitCommit=${GIT_COMMIT}' \
    -X 'main.BuildTime=${BUILD_TIME}'" \
    -o whoami .

# Runtime stage
FROM ysicing/debian

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/whoami .

# Create directory for configmaps
RUN mkdir -p /etc/config

# Expose port
EXPOSE 8080

# Start the application
CMD ["./whoami"]
