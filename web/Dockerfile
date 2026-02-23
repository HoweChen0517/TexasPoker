# syntax=docker/dockerfile:1

FROM node:20-alpine AS build
WORKDIR /app

COPY web/package*.json ./
RUN npm ci

COPY web/ ./

# Pass your backend URL at build time:
# docker build --build-arg VITE_API_URL=https://api.example.com -f web/Dockerfile .
ARG VITE_API_URL
ENV VITE_API_URL=${VITE_API_URL}

RUN npm run build

FROM nginx:1.27-alpine
COPY --from=build /app/dist /usr/share/nginx/html

EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
