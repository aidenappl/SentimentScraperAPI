# Stage 1: Build
FROM golang:1.23-alpine AS builder

# Install git for fetching dependencies (e.g., from GitHub)
RUN apk add --no-cache git

# Set the working directory inside the container
WORKDIR /app

# Copy module files and download dependencies (better layer caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the Go binary with size optimizations
RUN go build -ldflags="-w -s" -o /bin/app

# Stage 2: Minimal runtime container
FROM alpine:latest

# Create a working directory
WORKDIR /app

# Copy the binary
COPY --from=builder /bin/app .

# Copy the VADER lexicon files (adjust paths if needed)
COPY --from=builder /app/vader_lexicon.txt .
COPY --from=builder /app/emoji_utf8_lexicon.txt .

# Run the app
ENTRYPOINT ["./app"]
