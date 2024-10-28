# Notes

Make sure not to use any html settings that cause the form to reload.
This screws up the process of data entry, and displaying info.
e.g. the default behaviour of a submit button will reload the page and cause the wasm module to restart.


## Build

go build -o ..\web-server\main.wasm




```yml
# Use an official Go runtime as a parent image
FROM golang:1.17 AS builder

# Set the working directory in the container
WORKDIR /app

# Copy the current directory contents into the container at /app
COPY . .

# Build the WASM client
RUN GOOS=js GOARCH=wasm go build -o main.wasm main.go
```



```yml
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
#WORKDIR /

# Copy the built WASM binary from the builder stage
COPY --from=builder /out/main.wasm .

# Set GOROOT explicitly and copy the `wasm_exec.js` support file
# We assume Go is installed in the standard location on Alpine
ENV GOROOT=/usr/local/go
COPY --from=builder $GOROOT/misc/wasm/wasm_exec.js .

# You can now serve the main.wasm and wasm_exec.js as static files
```



## Form validation

<https://html.spec.whatwg.org/multipage/form-control-infrastructure.html#the-constraint-validation-api>



### Events

An event function in go uses the following construct:

```go
func SubmitItemEdit(this js.Value, args []js.Value) interface{} {
    event := args[0] // This provides access to the event object
    info := event.Get("type").String() // This provide the event type
}
```

