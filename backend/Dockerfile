# === Build stage: используем полноценный Debian-образ ===
FROM golang:1.22.5-bookworm AS builder

WORKDIR /app

# Копируем manifest-файлы
COPY go.mod go.sum ./

# Принудительно фиксируем версию Go в модуле (ключевой фикс!)
RUN go mod edit -go=1.22

# Скачиваем зависимости
RUN go mod download

# Копируем исходный код
COPY . .

# Собираем бинарник
RUN CGO_ENABLED=0 GOOS=linux go build -o /todo-server main.go

# === Run stage: минимальный Debian ===
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y ca-certificates curl && rm -rf /var/lib/apt/lists/*

WORKDIR /root/
COPY --from=builder /todo-server .

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --retries=3 \
  CMD curl -f http://localhost:8080/health || exit 1

CMD ["./todo-server"]
