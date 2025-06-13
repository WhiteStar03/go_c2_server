
FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY config/ ./config/
COPY controllers/ ./controllers/
COPY database/ ./database/
COPY middleware/ ./middleware/
COPY models/ ./models/
COPY routes/ ./routes/
COPY main.go .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /app/server .

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app
COPY --from=builder /app/server .

# Create binaries directory that will be mounted as volume
RUN mkdir -p /app/binaries

EXPOSE 8080
CMD ["./server"]