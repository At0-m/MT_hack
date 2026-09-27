FROM node:24-bookworm-slim AS build
WORKDIR /app
RUN corepack enable && corepack prepare pnpm@10.12.1 --activate
COPY package.json pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile
COPY . .
# The production image cannot silently enable the in-browser mock.
ENV VITE_API_MODE=real
RUN pnpm build

FROM nginxinc/nginx-unprivileged:1.28-alpine AS runtime
USER 101:101
COPY --from=build --chown=101:101 /app/dist /usr/share/nginx/html
COPY --chown=101:101 deploy/nginx.conf /etc/nginx/nginx.conf
EXPOSE 8080
HEALTHCHECK --interval=5s --timeout=3s --start-period=5s --retries=20 CMD wget -q -O /dev/null http://127.0.0.1:8080/_frontend_health || exit 1
ENTRYPOINT ["nginx", "-g", "daemon off;"]
