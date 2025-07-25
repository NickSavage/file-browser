# Build stage for React frontend
FROM node:18-alpine AS web-builder

WORKDIR /app/web
COPY web/package*.json ./
RUN npm install

COPY web/ ./
RUN npm run build

# Build stage for Go backend
FROM golang:1.21-alpine AS go-builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# Final stage
FROM alpine:latest

# Install ca-certificates
RUN apk --no-cache add ca-certificates
WORKDIR /root/

# Copy the Go binary
COPY --from=go-builder /app/main .

# Copy the React build
COPY --from=web-builder /app/web/build ./web/build

# Create data directory
RUN mkdir -p /data

# Set environment variables
ENV SERVE_DIR=/data
ENV PORT=8080
ENV JWT_SECRET=change-this-in-production
ENV ADMIN_PASSWORD=admin123

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/ || exit 1

CMD ["./main"]