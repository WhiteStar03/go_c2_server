############################
# Stage 1: Build React App #
############################
FROM node:18-alpine AS frontend-builder

# 1. Set working dir for frontend build
WORKDIR /app/c2client

# 2. Copy only manifest files for cached install
COPY c2client/package.json c2client/package-lock.json ./

# 3. Install JS dependencies
RUN npm ci

# 4. Copy rest of the React source
COPY c2client/ .

# 5. Build production assets into "build/"
RUN npm run build


############################
# Stage 2: Build Go Server #
############################
FROM golang:1.20-alpine AS backend-builder

# 6. Install git (for go modules) and CA certs
RUN apk add --no-cache git ca-certificates

# 7. Set working dir for Go
WORKDIR /app

# 8. Copy Go module declarations for cached deps
COPY go.mod go.sum ./

# 9. Download Go module dependencies
RUN go mod download

# 10. Copy entire repo (including /binaries)
COPY . .

# 11. Compile static, stripped binary
RUN CGO_ENABLED=0 \
    GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o server main.go


##################################
# Stage 3: Assemble Final Image  #
##################################
FROM alpine:latest

# 12. Ensure HTTPS (if your server makes outbound calls)
RUN apk add --no-cache ca-certificates

# 13. Set runtime working dir
WORKDIR /app

# 14. Copy the Go server binary
COPY --from=backend-builder /app/server .

# 15. Copy your pre-compiled placeholders in /binaries
COPY --from=backend-builder /app/binaries ./binaries

# 16. Copy React static files
COPY --from=frontend-builder /app/c2client/build ./static

# 17. Expose the port the server listens on
EXPOSE 8080

# 18. Run Gin in release mode
ENV GIN_MODE=release

# 19. Launch your server (which should serve both /api and ./static)
CMD ["./server"]
