# File Browser

A modern, fast file browser with web interface built with Go backend and React frontend. Features file indexing for fast browsing, upload/download capabilities, and Docker deployment.

## Features

- 🚀 **Fast Browsing**: Pre-indexed file system for instant navigation
- 📁 **Full File Operations**: Upload, download, rename, delete files and folders
- 🔍 **Search**: Quick file search within directories
- 🎨 **Modern UI**: Clean React interface with grid and list views
- 🐳 **Docker Ready**: Easy deployment with Docker and docker-compose
- 🔒 **Secure**: Path traversal protection and safe file operations

## Quick Start with Docker

### Using Pre-built Image (Recommended)

1. Download the production compose file:
   ```bash
   wget https://raw.githubusercontent.com/your-repo/file-browser/main/docker-compose.prod.yml
   # or create docker-compose.yml with the content below
   ```

2. Update `docker-compose.prod.yml` to mount your directory:
   ```yaml
   volumes:
     - /your/actual/directory:/data:ro  # Change this path
   ```

3. Run:
   ```bash
   docker-compose -f docker-compose.prod.yml up -d
   ```

### Building from Source

1. Clone the repository:
   ```bash
   git clone <repository-url>
   cd file-browser
   ```

2. Update `docker-compose.yml` to mount your directory:
   ```yaml
   volumes:
     - /your/actual/directory:/data:ro  # Change this path
   ```

3. Build and run:
   ```bash
   docker-compose up --build
   ```

4. Open http://localhost:8080 in your browser

## Development

### Prerequisites
- Go 1.21+
- Node.js 18+
- npm

### Running Locally

1. **Start the backend:**
   ```bash
   # Install Go dependencies
   go mod tidy
   
   # Set the directory to serve (optional, defaults to ./data)
   export SERVE_DIR=/path/to/your/files
   
   # Run the Go server
   go run main.go
   ```

2. **Start the frontend (in development mode):**
   ```bash
   cd web
   npm install
   npm start
   ```

3. Open http://localhost:3000 for development (with hot reload)

### Building for Production

```bash
# Build React frontend
cd web
npm install
npm run build
cd ..

# Build Go binary
go build -o file-browser main.go

# Set environment and run
export SERVE_DIR=/path/to/your/files
./file-browser
```

## Docker Hub Deployment

### Building and Pushing

1. Update the `DOCKER_USERNAME` in the Makefile with your Docker Hub username
2. Build and push:
   ```bash
   make build    # Build the image
   make push     # Push to Docker Hub
   make release  # Build and push with version tags
   ```

### Server Deployment

On your server, create a `docker-compose.yml` file:
```yaml
version: '3.8'
services:
  file-browser:
    image: your-dockerhub-username/file-browser:latest
    ports:
      - "8080:8080"
    volumes:
      - /your/server/files:/data:ro
    environment:
      - SERVE_DIR=/data
    restart: unless-stopped
```

Then run: `docker-compose up -d`

## Configuration

### Environment Variables

- `SERVE_DIR`: Directory to serve files from (default: `./data`)
- `PORT`: Server port (default: `8080`)

### Docker Volume Mounting

The Docker container expects files to be mounted at `/data`. Update the docker-compose.yml volume mapping:

```yaml
volumes:
  - /host/path/to/files:/data:ro  # :ro for read-only (recommended)
  # or
  - /host/path/to/files:/data     # for read-write access
```

## API Endpoints

- `GET /api/index` - Get full file index
- `POST /api/index/rebuild` - Rebuild file index
- `GET /api/browse/*path` - Browse directory contents
- `GET /api/download/*path` - Download file
- `POST /api/upload/*path` - Upload file
- `PUT /api/rename/*path` - Rename file/folder
- `DELETE /api/delete/*path` - Delete file/folder
- `POST /api/mkdir/*path` - Create directory

## Security Notes

- The application includes path traversal protection
- Consider using read-only mounts (`:ro`) if you only need browsing/downloading
- The container runs as non-root user for security
- Files are served only from the configured `SERVE_DIR`

## License

MIT License