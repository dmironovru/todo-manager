#!/bin/bash

echo "🛑 Stopping Todo Manager..."

# Останавливаем по PID
if [ -f /tmp/todo-backend.pid ]; then
    kill $(cat /tmp/todo-backend.pid) 2>/dev/null
    rm /tmp/todo-backend.pid
fi

if [ -f /tmp/todo-frontend.pid ]; then
    kill $(cat /tmp/todo-frontend.pid) 2>/dev/null
    rm /tmp/todo-frontend.pid
fi

# Добиваем остатки
pkill -f "go run cmd/api/main.go" 2>/dev/null
pkill -f "npm start" 2>/dev/null

# Останавливаем Docker
docker-compose down 2>/dev/null

# Освобождаем порты
fuser -k 3000/tcp 2>/dev/null
fuser -k 8080/tcp 2>/dev/null

echo "✅ All services stopped!"
