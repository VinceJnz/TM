* Notes



```go
            document.getElementById("fetchDataBtn").addEventListener("click", () => {
                const userData = fetchUserData();
                userData.then(data => {
                    const user = JSON.parse(data);
                    document.getElementById("output").innerHTML = `
                        <p>ID: ${user.id}</p>
                        <p>Name: ${user.name}</p>
                        <p>Username: ${user.username}</p>
                        <p>Email: ${user.email}</p>
                    `;
                });
            });

        });
```



```yml
# Use an official Go runtime as a parent image
FROM golang:1.17 AS builder

# Set the working directory in the container
WORKDIR /app

# Copy the current directory contents into the container at /app
COPY . .

# Build the webserver
RUN CGO_ENABLED=0 GOOS=linux go build -o main .
```


```yml
# Use a lightweight Go image
FROM golang:1.21.10-alpine
##FROM golang:1.20-alpine

# Set working directory in the container
WORKDIR /app

# Copy the Go web server source code to the container
COPY . .

# Copy the built WASM files to the static directory
#COPY ./static ./static
COPY ./static .

# Build the Go web server
RUN go mod download
RUN go build -o webserver .

# Expose the port the web server listens on
EXPOSE 8081

# Command to run the web server
CMD ["./webserver"]
```



```yml
# Use an official Go runtime as a parent image
FROM golang:1.17 AS builder

# Set the working directory in the container
WORKDIR /app

# Copy the current directory contents into the container at /app
COPY . .

# Build the webserver
RUN CGO_ENABLED=0 GOOS=linux go build -o main .
```