# Use a lightweight Go image
FROM golang:1.21.10-alpine
##FROM golang:1.20-alpine

# Set working directory in the container
WORKDIR /app

# Copy the Go web server source code to the container
COPY . .

# Copy the built WASM files to the static directory
COPY ./static ./static

# Build the Go web server
RUN go mod download
RUN go build -o webserver .

# Expose the port the web server listens on
EXPOSE 8081

# Command to run the web server
CMD ["./webserver"]
