# Use the official Golang image as a build stage
FROM golang:1.23 AS builder
WORKDIR /app
COPY . .

# Download dependencies
RUN go mod download

# Compile with static linking
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# Use a minimal image for the final stage
FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/main .
COPY --from=builder /app/.env .
EXPOSE 9876
CMD ["./main"]
