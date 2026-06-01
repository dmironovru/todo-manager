# 🚀 Todo Manager

Простой менеджер задач с ограничением на 10 задач.

**Демо:** [https://dmitrymironov.ru/lab/](https://dmitrymironov.ru/lab/)

## 🛠 Стек
- 🎨 Frontend: React 19 (Create React App)
- 🐹 Backend: Go 1.22 + PostgreSQL 16
- 🐳 Опционально: Docker Compose

## 🚀 Быстрый старт

### Вариант А: Через Docker
```bash
git clone <repo> && cd todo-manager
docker compose up -d
# Открой: 🌐 http://localhost:3000/lab


FROM node:20-alpine AS builder
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
ENV PUBLIC_URL=/lab
RUN npm run build

FROM nginx:alpine
COPY --from=builder /app/build /usr/share/nginx/html/lab
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
