# Use a lightweight Go image to build the WASM binary
FROM golang:1.21.10-alpine AS builder
##FROM golang:1.20-alpine as builder

# Set the working directory inside the container
WORKDIR /app

# Copy the Go source code into the container
COPY . .

# Build the Go WASM binary
RUN GOOS=js GOARCH=wasm go build -o /out/main.wasm

# Stage 2: Create a clean image for the final output
FROM golang:1.21.10-alpine
##FROM alpine:3.18

# Set the working directory in the final container
WORKDIR /static

# Copy the built WASM binary from the builder stage
COPY --from=builder /out/main.wasm .

# Set GOROOT explicitly and copy the `wasm_exec.js` support file
# We assume Go is installed in the standard location on Alpine
ENV GOROOT=/usr/local/go
COPY --from=builder $GOROOT/misc/wasm/wasm_exec.js .

# You can now serve the main.wasm and wasm_exec.js as static files