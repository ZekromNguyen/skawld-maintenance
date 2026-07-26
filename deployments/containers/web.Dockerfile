FROM docker.io/library/node:22.22.0-bookworm-slim AS build
WORKDIR /src
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM docker.io/nginxinc/nginx-unprivileged:1.29-alpine
COPY deployments/pilot/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /src/dist /usr/share/nginx/html
EXPOSE 8080
