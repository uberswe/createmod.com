# Dockerfile for CreateMod application with PocketBase and frontend

# Build stage for frontend
FROM node:20-alpine AS frontend-builder
WORKDIR /app
COPY ./template/package*.json ./
RUN npm install
COPY ./template ./
RUN npm run build

# Build stage for Go application
FROM golang:alpine AS go-builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Ensure pb_public directory exists
RUN mkdir -p pb_public
RUN go build -o main ./cmd/server/main.go

# Final stage
FROM alpine:latest

LABEL maintainer="Uberswe <admin@uberswe.com>"
LABEL org.opencontainers.image.source="https://github.com/uberswe/createmod"
LABEL org.opencontainers.image.licenses=MIT

# Install required packages
RUN apk add --update --no-cache ca-certificates curl git openssl sqlite tar tzdata \
    && adduser -D -h /home/container container

USER container
ENV USER=container HOME=/home/container
WORKDIR /home/container

# Copy built artifacts from previous stages
COPY --from=frontend-builder /app/dist /home/container/template/dist
COPY --from=go-builder /app/main /home/container/main
COPY --from=go-builder /app/template /home/container/template
COPY --from=go-builder /app/pb_public /home/container/pb_public

# Create necessary directories
RUN mkdir -p /home/container/pb_data /home/container/migrations

# Copy entrypoint script
COPY ./pterodactyl/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

EXPOSE 8090

CMD ["/bin/sh", "/entrypoint.sh"]
