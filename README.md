* Notes






```yml
version: '3.8'

services:
  wasm-builder:
    build:
      context: .
      dockerfile: Dockerfile.wasm
    volumes:
      - .:/app
    depends_on:
      - webserver-builder

  webserver-builder:
    build:
      context: .
      dockerfile: Dockerfile.webserver
    volumes:
      - .:/app
    depends_on:
      - wasm-builder

  webserver:
    build:
      context: .
      dockerfile: Dockerfile
    ports:
      - "8080:80"
    depends_on:
      - wasm-builder
      - webserver-builder
```



```yml
  wasmclient:
    build:
      context: ./client1
    container_name: wasmclient
    networks:
      - mynetwork
    volumes:
      - ./webserver/static:/static
##    depends_on:
##      - webserver

  webserver:
    build:
      context: ./web-server        # Updated context to web-server subfolder
##      dockerfile: Dockerfile-web   # Dockerfile for the web server
    container_name: webserver
    ports:
      - "8081:8081"
    networks:
      - mynetwork
    volumes:
      - ./web-server/static:/app/static
    depends_on:
##      - apiserver
      - wasmclient
```

