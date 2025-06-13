# go_backend/Dockerfile

# Stage 1: Build the Go application
FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

# Copy source code (exclude binaries directory)
COPY config/ ./config/
COPY controllers/ ./controllers/
COPY database/ ./database/
COPY middleware/ ./middleware/
COPY models/ ./models/
COPY routes/ ./routes/
COPY main.go .

# Build the Go application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /app/server .

# Stage 2: Create the final small image
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app
COPY --from=builder /app/server .

# Create binaries directory that will be mounted as volume
RUN mkdir -p /app/binaries

EXPOSE 8080
CMD ["./server"]