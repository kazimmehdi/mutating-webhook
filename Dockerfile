FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY main.go ./

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o webhook main.go

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/


# Copy the binary from builder
COPY --from=builder /app/webhook /usr/local/bin/webhook


# Copy the binary with the correct ownership
# COPY  ./webhook /usr/local/bin/webhook


CMD ["/usr/local/bin/webhook"]
