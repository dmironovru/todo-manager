#!/bin/bash
set -e

PROJECT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LOG_DIR="$PROJECT_DIR/logs"
mkdir -p "$LOG_DIR"

GREEN='\033[0;32m'; BLUE='\033[0;34m'; RED='\033[0;31m'; YELLOW='\033[0;33m'; NC='\033[0m'
log() { echo -e "${BLUE}[Todo]${NC} $1"; }
info() { echo -e "${GREEN}[✓]${NC} $1"; }
warn() { echo -e "${YELLOW}[!]${NC} $1"; }
error() { echo -e "${RED}[✗]${NC} $1"; }

cleanup() {
    log "Остановка..."
    [ -n "$BACKEND_PID" ] && kill $BACKEND_PID 2>/dev/null || true
    [ -n "$FRONTEND_PID" ] && kill $FRONTEND_PID 2>/dev/null || true
    info "Готово! 👋"
    exit 0
}
trap cleanup SIGINT SIGTERM

# === ПРОВЕРКА НАЛИЧИЯ DOCKER ===
log "Проверка Docker..."
if ! command -v docker >/dev/null 2>&1; then
    error "Docker не установлен!"
    echo ""
    echo -e "${YELLOW}🔧 Установи Docker:${NC}"
    echo -e "   ${GREEN}curl -fsSL https://get.docker.com | sh${NC}"
    echo -e "   ${GREEN}sudo usermod -aG docker \$USER${NC}"
    echo -e "   ${GREEN}newgrp docker${NC}"
    echo ""
    echo -e "${YELLOW}📖 Или скачай Docker Desktop:${NC}"
    echo -e "   https://www.docker.com/products/docker-desktop/"
    exit 1
fi
info "Docker установлен ✅"

# === ПРОВЕРКА ЗАПУЩЕН ЛИ DOCKER ===
if ! docker info >/dev/null 2>&1; then
    error "Docker не запущен!"
    echo ""
    echo -e "${YELLOW}🔧 Запусти Docker:${NC}"
    echo -e "   ${GREEN}sudo systemctl start docker${NC}"
    echo -e "   ${GREEN}sudo systemctl enable docker${NC}"
    echo ""
    echo -e "${YELLOW}💡 Или открой Docker Desktop${NC}"
    exit 1
fi
info "Docker запущен ✅"

# === ПРОВЕРКА ПРАВ DOCKER ===
if ! docker ps >/dev/null 2>&1; then
    warn "Недостаточно прав для Docker!"
    
    if groups $USER | grep -q docker; then
        warn "Вы уже в группе docker, но изменения не применились."
        echo ""
        echo -e "${YELLOW}🔄 Выполните:${NC}"
        echo -e "   ${GREEN}newgrp docker${NC}"
        echo -e "   ${GREEN}./start.sh${NC}"
        exit 1
    else
        warn "Добавляем пользователя $USER в группу docker..."
        sudo usermod -aG docker $USER
        echo ""
        info "Пользователь добавлен в группу docker ✅"
        echo ""
        echo -e "${YELLOW}🔄 Выполните:${NC}"
        echo -e "   ${GREEN}newgrp docker${NC}"
        echo -e "   ${GREEN}./start.sh${NC}"
        exit 0
    fi
fi
info "Права Docker OK ✅"

# === ЗАПУСК ===
if [ -f "docker-compose.yml" ]; then
    log "Запуск через Docker Compose..."
    docker compose up -d
    echo ""
    echo -e "${GREEN}╔══════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║  🚀 Todo Manager запущен!               ║${NC}"
    echo -e "${GREEN}╠══════════════════════════════════════════╣${NC}"
    echo -e "${GREEN}║  🌐 Открой в браузере:                  ║${NC}"
    echo -e "${GREEN}║     http://localhost:3000               ║${NC}"
    echo -e "${GREEN}║                                         ║${NC}"
    echo -e "${GREEN}║  🛑 Остановка:                          ║${NC}"
    echo -e "${GREEN}║     docker compose down                 ║${NC}"
    echo -e "${GREEN}║     или ./stop.sh                      ║${NC}"
    echo -e "${GREEN}╚══════════════════════════════════════════╝${NC}"
    exit 0
fi

# === НАТИВНЫЙ ЗАПУСК (без Docker) ===
log "Нативный запуск..."
command -v go >/dev/null || { error "Go не найден"; exit 1; }
command -v node >/dev/null || { error "Node не найден"; exit 1; }

cd "$PROJECT_DIR/backend"
export DATABASE_URL="postgres://todo_user:todo_pass@localhost:5432/todo_db"
go run main.go > "$LOG_DIR/backend.log" 2>&1 &
BACKEND_PID=$!
sleep 2
info "Бэкенд на порту 8080 ✅"

cd "$PROJECT_DIR/frontend"
[ ! -d "node_modules" ] && npm install --silent
npm run start > "$LOG_DIR/frontend.log" 2>&1 &
FRONTEND_PID=$!
sleep 3
info "Фронтенд на порту 3000 ✅"

echo ""
echo -e "${GREEN}╔══════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║  🚀 Todo Manager запущен!               ║${NC}"
echo -e "${GREEN}╠══════════════════════════════════════════╣${NC}"
echo -e "${GREEN}║  🌐 Открой в браузере:                  ║${NC}"
echo -e "${GREEN}║     http://localhost:3000               ║${NC}"
echo -e "${GREEN}║                                         ║${NC}"
echo -e "${GREEN}║  🛑 Остановка:                          ║${NC}"
echo -e "${GREEN}║     Нажми Ctrl+C                        ║${NC}"
echo -e "${GREEN}╚══════════════════════════════════════════╝${NC}"

while kill -0 $BACKEND_PID $FRONTEND_PID 2>/dev/null; do sleep 5; done
cleanup