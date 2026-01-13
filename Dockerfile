# Stage 1: Build the application
FROM golang:1.21.13-alpine AS builder

# Set GOPROXY (using default Go proxy)
ENV GOPROXY=https://proxy.golang.org,direct

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies with timeout
RUN timeout -s INT 60s go mod download || go mod download

# Copy the source code
COPY . .

# Build the application
RUN go build -o triggermesh ./cmd/triggermesh

# Stage 2: Create the final image
FROM alpine:3.18

WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/triggermesh .

# Copy the default configuration file
COPY config.yaml.example ./config.yaml.example

# Expose the port the app runs on
EXPOSE 8080

# Command to run the application
CMD ["./triggermesh", "--config", "./config.yaml"]
