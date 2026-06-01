#!/bin/bash
set -e

PROJECT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LOG_DIR="$PROJECT_DIR/logs"
mkdir -p "$LOG_DIR"

GREEN='\033[0;32m'; BLUE='\033[0;34m'; NC='\033[0m'
log() { echo -e "${BLUE}[Todo]${NC} $1"; }
info() { echo -e "${GREEN}[✓]${NC} $1"; }

cleanup() {
    log "Остановка..."
    [ -n "$BACKEND_PID" ] && kill $BACKEND_PID 2>/dev/null || true
    [ -n "$FRONTEND_PID" ] && kill $FRONTEND_PID 2>/dev/null || true
    info "Готово! 👋"
    exit 0
}
trap cleanup SIGINT SIGTERM

# Если есть Docker — используем его
if command -v docker >/dev/null 2>&1 && [ -f "docker-compose.yml" ]; then
    log "Запуск через Docker..."
    docker compose up -d
    echo ""
    echo -e "${GREEN}╔════════════════════════╗${NC}"
    echo -e "${GREEN}║  🚀 Todo запущен!       ║${NC}"
    echo -e "${GREEN}╠════════════════════════╣${NC}"
    echo -e "${GREEN}║  🌐 http://localhost:3000/lab  ║${NC}"
    echo -e "${GREEN}║  🔌 http://localhost:8080      ║${NC}"
    echo -e "${GREEN}║  🛑 docker compose down        ║${NC}"
    echo -e "${GREEN}╚════════════════════════╝${NC}"
    exit 0
fi

# Иначе — нативный запуск
log "Нативный запуск..."
command -v go >/dev/null || { echo "❌ Go не найден"; exit 1; }
command -v node >/dev/null || { echo "❌ Node не найден"; exit 1; }

# Запуск бэкенда
cd "$PROJECT_DIR/backend"
export DATABASE_URL="postgres://todo_user:todo_pass@localhost:5432/todo_db"
go run main.go > "$LOG_DIR/backend.log" 2>&1 &
BACKEND_PID=$!
sleep 2
info "Бэкенд на порту 8080 ✅"

# Запуск фронтенда
cd "$PROJECT_DIR/frontend"
[ ! -d "node_modules" ] && npm install --silent
npm run start > "$LOG_DIR/frontend.log" 2>&1 &
FRONTEND_PID=$!
sleep 3
info "Фронтенд на порту 3000 ✅"

echo ""
echo -e "${GREEN}╔════════════════════════╗${NC}"
echo -e "${GREEN}║  🚀 Todo запущен!       ║${NC}"
echo -e "${GREEN}╠════════════════════════╣${NC}"
echo -e "${GREEN}║  🌐 http://localhost:3000/lab  ║${NC}"
echo -e "${GREEN}║  💡 Ctrl+C для остановки ║${NC}"
echo -e "${GREEN}╚════════════════════════╝${NC}"

while kill -0 $BACKEND_PID $FRONTEND_PID 2>/dev/null; do sleep 5; done
cleanup
