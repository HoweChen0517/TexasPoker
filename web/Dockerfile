# syntax=docker/dockerfile:1.7

FROM --platform=$BUILDPLATFORM node:20-alpine AS build
WORKDIR /app

ARG VERSION=dev

COPY web/package*.json ./
RUN --mount=type=cache,target=/root/.npm npm ci

COPY web/ ./

ARG VITE_API_URL=http://localhost:8080
ENV VITE_API_URL=$VITE_API_URL

RUN npm run build

FROM nginx:1.27-alpine
ARG VERSION=dev
LABEL org.opencontainers.image.title="texaspoker-web"
LABEL org.opencontainers.image.version=$VERSION
LABEL org.opencontainers.image.source="https://github.com/howechen/TexasPoker"

COPY web/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/dist /usr/share/nginx/html

EXPOSE 80
ENTRYPOINT ["nginx", "-g", "daemon off;"]
