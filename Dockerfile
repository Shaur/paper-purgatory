# Use the official Golang image as the base image
FROM golang:1.25.1-alpine3.22 AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum files to the working directory
COPY go.mod go.sum ./

# Download all dependencies
RUN go mod download

# Copy the source code to the working directory
COPY . ./

ENV CGO_ENABLED=1

RUN apk add --no-cache build-base linux-headers

# Build the Go application
RUN go build -o purgatory .

# Use a minimal base image for the final container
FROM alpine:3.22

# Copy the built binary from the builder stage
COPY --from=builder /app/purgatory purgatory

RUN chmod +x purgatory

COPY application.yaml application.yaml

# Expose port 8080 for the application
EXPOSE 8080

# Command to run the application
CMD ["./purgatory"]