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
