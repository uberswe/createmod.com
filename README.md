# CreateMod.com
[![deploy](https://github.com/uberswe/createmod.com/actions/workflows/deploy.yml/badge.svg)](https://github.com/uberswe/createmod.com/actions/workflows/deploy.yml)

This repository contains all the files needed to run CreateMod.com

## Architecture

CreateMod.com uses a decoupled architecture with:
- **Backend**: PocketBase (Go) providing a REST API and file storage
- **Frontend**: Next.js (React) with Server-Side Rendering (SSR)

This architecture allows for:
- Independent development of frontend and backend
- Better performance through SSR and client-side navigation
- Improved developer experience with hot reloading
- Maintaining the same URL structure as the previous version

## Development Setup

### Backend

To run the backend server:

```bash
# Start the PocketBase server
go run ./cmd/server/main.go serve
```

The backend API will be available at http://localhost:8090/api/

### Frontend

To run the Next.js development server:

```bash
# Navigate to the frontend directory
cd ./frontend

# Install dependencies (first time only)
npm install

# Start the development server
npm run dev
```

The frontend will be available at http://localhost:3000

### Full Stack Development

For full stack development, you'll need to run both the backend and frontend servers simultaneously (in separate terminal windows).

## Production Build

To build the application for production:

### Backend

The backend is a Go application that can be built with:

```bash
go build -o createmod cmd/server/main.go
```

### Frontend

To build the Next.js frontend for production:

```bash
# Navigate to the frontend directory
cd ./frontend

# Install dependencies (if not already installed)
npm install

# Build for production
npm run build

# Optional: Start the production server
npm run start
```

The production build will generate static files and server components in the `frontend/.next` directory.

## Deployment

For deployment, you have several options:

1. **Traditional Deployment**:
   - Deploy the Go backend on your server
   - Deploy the Next.js frontend using a Node.js server

2. **Static Export** (if not using dynamic routes):
   - Build the Next.js app with `npm run build`
   - Export static files with `npx next export`
   - Serve the static files with any web server

3. **Containerized Deployment**:
   - Use the provided Docker setup (see below)

## Docker

[Docker](https://www.docker.com/) is provided to make development and deployment easier. The Docker setup includes both the backend and frontend services.

```bash
# Start both backend and frontend
docker compose up

# Rebuild containers if you make changes
docker compose up --build
```

The application will be available at:
- Backend API: [http://localhost:8090/api](http://localhost:8090/api)
- Frontend: [http://localhost:3000](http://localhost:3000)

### Docker Services

The docker-compose.yml file defines the following services:
- **backend**: The PocketBase server
- **frontend**: The Next.js frontend
- **npm**: A service for running npm commands in the frontend directory

## Environment Variables

### Backend Environment Variables

These environment variables are used by the PocketBase backend:

#### Auto Migrate

Auto Migrate can be used to automatically generate database migration files when changes to the data structures are made.

```
AUTO_MIGRATE=true
```

#### Create Admin

If Create Admin is set to true an admin is generated. This is convenient for local development.

```
CREATE_ADMIN=true
```

The default credentials are `local@createmod.com` and `jfq.utb*jda2abg!WCR`. Do not use these credentials in a live environment.

#### Dummy data

You can set the following to true to generate dummy data. Please note that it will only work when running the migrations for the first time. Delete the `pb_data` to reset. WARNING this deletes all data.

```
DUMMY_DATA=true
```

#### Discord Webhook URL

A webhook url can be specified for Discord notifications

```
DISCORD_WEBHOOK_URL=https://discord.com/api/webhooks/122...
```

You can read more about Discord webhooks here: https://discord.com/developers/docs/resources/webhook

### Frontend Environment Variables

The Next.js frontend uses environment variables for configuration. Create a `.env.local` file in the `frontend` directory with the following variables:

```
# Backend API URL (development)
NEXT_PUBLIC_API_URL=http://localhost:8090/api

# Backend API URL (production)
# NEXT_PUBLIC_API_URL=https://createmod.com/api

# Site URL
NEXT_PUBLIC_SITE_URL=http://localhost:3000
```

For production, you should set these variables in your deployment environment.

## Frontend-Backend Communication

The Next.js frontend communicates with the PocketBase backend in two ways:

### 1. Client-side API Calls

For client-side interactions (like submitting forms, rating schematics, etc.), the frontend makes direct API calls to the backend using the fetch API. These calls are authenticated using cookies.

### 2. Server-Side Rendering (SSR)

For initial page loads, the Next.js server fetches data from the PocketBase API during the server-side rendering process. This provides:

- Better SEO as pages are pre-rendered with content
- Faster initial page loads
- Proper authentication state on first load

The communication flow works like this:

1. User requests a page from the Next.js server
2. Next.js server makes API requests to PocketBase
3. Next.js renders the page with the data
4. The page is sent to the user's browser
5. Client-side JavaScript takes over for interactive elements

This architecture maintains the same URL structure as the previous version while providing a better user and developer experience.