# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

A modern file browser with Go backend and React frontend. Features fast file indexing, full CRUD operations (upload, download, rename, delete), and Docker deployment. The application indexes a configured directory on startup for fast browsing and serves files through a clean web interface.

## Development Commands

### Backend (Go)
```bash
# Install dependencies
go mod tidy

# Run the server (serves both API and built React app)
go run main.go

# Build binary
go build -o file-browser main.go

# Set custom serve directory
export SERVE_DIR=/path/to/files
go run main.go
```

### Frontend (React)
```bash
cd web

# Install dependencies
npm install

# Development server with hot reload (proxies API to :8080)
npm start

# Build for production
npm run build

# Run tests
npm test
```

### Docker
```bash
# Development: Build and run with docker-compose
docker-compose up --build

# Production: Build and push to Docker Hub
make build
make push
make release

# Server deployment with pre-built image
docker-compose -f docker-compose.prod.yml up -d
```

## Architecture

### Backend (Go - main.go)
- **Gin HTTP server** serving both API endpoints and static React build
- **File indexing system** that scans the configured directory on startup for fast browsing
- **REST API** for all file operations with path security validation 
- **CORS support** for development with React dev server

### Frontend (React - web/)
- **React Router** for client-side navigation (/browse/*)
- **Component structure**: FileBrowser, FileList, Header, FileUpload, etc.
- **Tailwind CSS** for styling with responsive design
- **Lucide React** icons for consistent UI
- **Grid and List views** for file browsing

### Key Components
- `main.go`: Main server with file indexing and API endpoints
- `web/src/components/FileBrowser.js`: Main browsing component with breadcrumbs
- `web/src/components/FileList.js`: File/folder display with grid/list modes
- `web/src/components/FileUpload.js`: Drag-and-drop upload modal
- `web/src/components/Header.js`: Top bar with index stats and rebuild button

### API Endpoints
- `GET /api/index` - Full file index with stats
- `GET /api/browse/*path` - Directory contents 
- `GET /api/download/*path` - File download
- `POST /api/upload/*path` - File upload
- `PUT /api/rename/*path` - Rename file/folder
- `DELETE /api/delete/*path` - Delete file/folder
- `POST /api/mkdir/*path` - Create directory
- `POST /api/index/rebuild` - Rebuild file index

### Security Features
- Path traversal protection (all paths validated against SERVE_DIR)
- File operations restricted to configured directory
- Safe file handling with proper error responses

## Environment Variables
- `SERVE_DIR`: Directory to serve (default: ./data) 
- `PORT`: Server port (default: 8080)

## Deployment Notes
- React app builds to `web/build/` and is served by Go server
- Docker multi-stage build: Node.js for React, Go for backend, Alpine for runtime
- Health checks included for container orchestration
- Supports read-only volume mounts for security