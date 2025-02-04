FROM golang:alpine

RUN apk update && apk upgrade && \
    apk add --no-cache bash git openssh

LABEL maintainer="Uberswe <admin@uberswe.com>"

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependancies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source from the current directory to the Working Directory inside the container
COPY . .

# CompileDemon for live reload
RUN go get github.com/githubnemo/CompileDaemon
RUN go install -mod=mod github.com/githubnemo/CompileDaemon

# Expose port 8080 to the outside world
EXPOSE 8090

# Run the executable

ENTRYPOINT CompileDaemon --build='go build -o main ./cmd/server/main.go' --command='./main serve --http 0.0.0.0:8090'