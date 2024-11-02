# Start with the official Go image
FROM golang:1.23-alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum, then download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
COPY . .

# Build the Go application
RUN go build -o api-sub-hunter main.go

# Start a new stage for the final image
FROM alpine:latest

# Set up a directory for the app
WORKDIR /app

# Copy the compiled Go binary and UI files from the builder stage
COPY --from=builder /app/api-sub-hunter /app/api-sub-hunter
COPY --from=builder /app/ui /app/ui

# Expose the port your app will listen on (assuming 8080)
EXPOSE 8080

# Command to run the application
CMD ["/app/api-sub-hunter"]
