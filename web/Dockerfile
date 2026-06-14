# syscert website — multi-stage build.
# Stage 1 builds the static Astro site; stage 2 serves it with nginx.
# Traefik terminates TLS in front of this container, so nginx only speaks HTTP:80.

# ---- build ----
FROM node:22-slim AS build
WORKDIR /app

# install deps against the lockfile first (better layer caching)
COPY package.json package-lock.json ./
RUN npm ci

# build the static site
COPY . .
RUN npm run build

# ---- serve ----
FROM nginx:alpine AS runtime
# drop nginx's default site, add ours
RUN rm /etc/nginx/conf.d/default.conf
COPY nginx.conf /etc/nginx/conf.d/syscert.conf
COPY --from=build /app/dist /usr/share/nginx/html

EXPOSE 80
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s \
  CMD wget -qO /dev/null http://127.0.0.1/ || exit 1
