FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /todo-server .

FROM alpine:latest
RUN apk --no-cache add ca-certificates curl
WORKDIR /root/
COPY --from=builder /todo-server .
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s --retries=3 CMD curl -f http://localhost:8080/health || exit 1
CMD ["./todo-server"]
