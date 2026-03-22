FROM node:24-alpine AS frontend-builder

WORKDIR /app/template

# Copy package files for dependency installation
COPY template/package.json ./

# Install dependencies fresh (no lock file) to get correct platform binaries
RUN npm install

# Copy template source files
COPY template/ ./

# Build frontend assets
RUN npm run build

FROM golang:alpine

RUN apk update && apk upgrade && \
    apk add --no-cache bash git openssh wget

LABEL maintainer="Uberswe <admin@uberswe.com>"

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependancies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source from the current directory to the Working Directory inside the container
COPY . .

# Copy built frontend assets into the static directory the server serves from
COPY --from=frontend-builder /app/template/dist/ ./template/static/

# Build the Go app
RUN go build -o main ./cmd/server/main.go

# Expose port 8080 to the outside world
EXPOSE 8080

# Run the executable
CMD ["./main"]
