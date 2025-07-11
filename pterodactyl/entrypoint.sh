#!/bin/sh
cd /home/container

# Output Current Working Directory
echo "Current working directory: $(pwd)"

# Make internal directory if it doesn't exist
if [ ! -d "./internal" ]; then
    mkdir -p ./internal
fi

# Make sure we have the required directories
mkdir -p ./pb_data ./migrations ./template/dist

# Default the TZ environment variable to UTC.
TZ=${TZ:-UTC}
export TZ

# Set environment variables from Pterodactyl
export AUTO_MIGRATE=${AUTO_MIGRATE:-false}
export CREATE_ADMIN=${CREATE_ADMIN:-false}
export DISCORD_WEBHOOK_URL=${DISCORD_WEBHOOK_URL:-""}

# Replace Startup Variables
MODIFIED_STARTUP=$(echo ${STARTUP} | sed -e 's/{{/${/g' -e 's/}}/}/g')
echo ":/home/container$ ${MODIFIED_STARTUP}"

# Run the Server
eval ${MODIFIED_STARTUP}