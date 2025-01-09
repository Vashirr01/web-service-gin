# Build stage
FROM golang:1.23 AS builder
WORKDIR /app
# Copy dependency files
COPY go.mod go.sum ./
RUN go mod download
# Copy the rest of the application
COPY . .
# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/main .

# Final stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
# Copy the binary from builder
COPY --from=builder /app/main .
# Copy the .env file
COPY .env .
# Expose the port your application runs on
EXPOSE 8080
CMD ["./main"]
