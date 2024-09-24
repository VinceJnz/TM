# Use a lightweight Go image
FROM golang:1.21.10-alpine
##FROM golang:1.20-alpine

# Set working directory in the container
WORKDIR /app

# Copy the Go API server source code to the container
COPY . .

# Build the Go API server
RUN go mod download
RUN go build -o apiserver .

# Expose the port the API server listens on
EXPOSE 8081

# Command to run the API server
CMD ["./apiserver"]
