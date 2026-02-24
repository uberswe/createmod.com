# Minimal PocketBase image built from official release archives
# See: https://pocketbase.io/docs/

FROM alpine:latest

ARG PB_VERSION=0.30.0

RUN apk add --no-cache \
    unzip \
    ca-certificates \
    wget

# Download and unzip PocketBase (linux amd64 build)
ADD https://github.com/pocketbase/pocketbase/releases/download/v${PB_VERSION}/pocketbase_${PB_VERSION}_linux_amd64.zip /tmp/pb.zip
RUN unzip /tmp/pb.zip -d /pb/

# Uncomment to copy local migrations/hooks into the image
# COPY ./dev/pb_migrations /pb/pb_migrations
# COPY ./dev/pb_hooks /pb/pb_hooks

EXPOSE 8090

# Entrypoint (port overridden by compose command)
CMD ["/pb/pocketbase", "serve", "--http=0.0.0.0:8090"]
