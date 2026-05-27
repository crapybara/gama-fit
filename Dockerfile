# ==========================================
# STAGE 1: Build the Go Binary
# ==========================================
FROM golang:1.21-alpine AS builder

# Install GCC and musl-dev (Required for SQLite CGO compilation)
RUN apk add --no-cache gcc musl-dev

# Set the working directory inside the container
WORKDIR /app

# Copy the Go module files and download dependencies
# (Doing this first caches the downloads to speed up future builds)
COPY internal/go.mod internal/go.sum ./internal/
RUN cd internal && go mod download

# Copy your entire project into the container
COPY . .

# Move into the internal directory to build the binary
WORKDIR /app/internal

# Build the binary with CGO enabled
RUN CGO_ENABLED=1 GOOS=linux go build -o gamafit-server main.go

# ==========================================
# STAGE 2: Create the Lightweight Runner
# ==========================================
FROM alpine:latest

# Add timezone data and certificates (good practice for servers)
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy the static HTML/CSS/JS frontend from the builder
COPY --from=builder /app/external ./external

# Move into internal to match your app's relative path logic ("../external")
WORKDIR /app/internal

# Copy the compiled binary from the builder
COPY --from=builder /app/internal/gamafit-server .

# Tell Docker what port this container listens on
EXPOSE 8080

# Boot the server
CMD ["./gamafit-server"]
