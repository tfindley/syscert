# syscert website — multi-stage build.
# Stage 1 builds the static Astro site; stage 2 serves it with nginx.
# Traefik terminates TLS in front of this container, so nginx only speaks HTTP:80.

# ---- build ----
FROM node:22-slim AS build
WORKDIR /app

# install deps against the lockfile first (better layer caching)
COPY package.json package-lock.json ./
RUN npm ci

# The tool release this site documents, injected by web.yml. Declared here (after
# npm ci, so the deps layer stays cached) so the version is a real build input: the
# `npm run build` layer's cache key changes per release, which busts buildx's gha
# cache and forces getRelease() to re-resolve — otherwise an unchanged source commit
# reuses a cached layer that baked the previous version. Defaults to "dev" for local builds.
ARG SITE_VERSION=dev
ENV SITE_VERSION=${SITE_VERSION}

# build the static site
COPY . .
RUN npm run build

# ---- serve ----
# alpine-slim (not alpine): we only serve static files with core/standard modules
# (see nginx.conf), so the optional dynamic modules — and their image-filter/libgd
# dependency chain (libtiff, freetype) — are pure attack surface. -slim drops them.
# Track the tag (not a pinned digest) so rebuilds pick up Alpine security patches.
FROM nginx:alpine-slim AS runtime
# drop nginx's default site, add ours
RUN rm /etc/nginx/conf.d/default.conf
COPY nginx.conf /etc/nginx/conf.d/syscert.conf
COPY --from=build /app/dist /usr/share/nginx/html

EXPOSE 80
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget -qO /dev/null http://127.0.0.1/healthz || exit 1
