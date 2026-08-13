# syntax=docker/dockerfile:1

# Stage bersama: install dependency sekali untuk dev dan build produksi.
FROM node:24-alpine AS base
RUN corepack enable
WORKDIR /app
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
RUN pnpm install --frozen-lockfile

# Stage dev: Vite dev server dengan HMR. Source code datang dari bind mount host,
# jadi perubahan langsung terlihat di browser tanpa build ulang image.
FROM base AS dev
ENV NODE_ENV=development
EXPOSE 5173
CMD ["pnpm", "exec", "vite", "--host", "0.0.0.0"]

# Stage build produksi/staging: static assets hasil `vite build`.
FROM base AS builder
COPY . .
RUN pnpm build

# Stage terakhir adalah default build target, sehingga `docker compose build` tanpa
# target tetap menghasilkan image Nginx berisi static assets.
FROM nginxinc/nginx-unprivileged:1.29-alpine AS runtime
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=builder /app/dist /usr/share/nginx/html
EXPOSE 8080
