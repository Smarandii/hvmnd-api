# Use the official Golang image as a build stage
FROM golang:1.23 AS builder
WORKDIR /app
COPY . .

# Download dependencies
RUN go mod download

# Compile with static linking
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# Use a lightweight Debian-based image for the final stage
FROM debian:bullseye-slim
WORKDIR /root/

# Install curl (debian-slim includes apt by default)
RUN apt-get update && apt-get install -y --no-install-recommends curl && rm -rf /var/lib/apt/lists/*

# Copy the built Go binary and environment file
COPY --from=builder /app/main .
COPY --from=builder /app/.env .

# Expose the application port
EXPOSE 9876

# Run the application
CMD ["./main"]
