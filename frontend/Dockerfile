FROM node:20-alpine AS builder
WORKDIR /app

ARG VITE_API_BASE
ARG VITE_WS_BASE
ENV VITE_API_BASE=${VITE_API_BASE}
ENV VITE_WS_BASE=${VITE_WS_BASE}

COPY package.json package-lock.json* ./
RUN npm install

COPY . ./
RUN npm run build

FROM nginx:1.27-alpine
COPY --from=builder /app/dist /usr/share/nginx/html

EXPOSE 80
