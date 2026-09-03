# syscert website — multi-stage build.
# Stage 1 builds the static Astro site; stage 2 serves it with nginx.
# Traefik terminates TLS in front of this container, so nginx only speaks HTTP,
# on :8080 — the whole process tree runs unprivileged (see the serve stage).

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

# Run fully unprivileged (same pattern as fitsd-web). The stock image starts its
# master as root and only drops WORKERS to uid 101, so a master-process compromise
# starts with root. Here the whole process tree runs as 101 and the container binds
# 8080 — no NET_BIND_SERVICE, no root anywhere.
#   • drop `user nginx;` — it is meaningless (and warns) without super-user privileges
#   • move the pid file to /tmp; the stock path is under root-owned /var/run
#   • PRE-CREATE the temp dirs. nginx builds them at startup with mkdir(), which fails
#     unprivileged — but ngx_create_paths() tolerates EEXIST and skips the chown when
#     no `user` directive is set, so pre-created dirs need no write at all. A static,
#     GET/HEAD-only server never writes a body temp file afterwards.
#
# The pid is REWRITTEN IN nginx.conf, not passed as `nginx -g "pid …"`: a -g directive
# is PREPENDED to the main context, so with the stock `pid` line still present nginx
# dies at startup with a duplicate-directive error. The grep FAILS THE BUILD if the
# substitution didn't match, so a future base-image change can't silently leave the
# pid in root-owned /var/run and turn this into a runtime crash.
RUN rm /etc/nginx/conf.d/default.conf \
 && sed -i \
      -e 's/^user .*/# no `user` directive: this image runs unprivileged (see USER below)/' \
      -e 's|^pid .*|pid /tmp/nginx.pid;|' \
      /etc/nginx/nginx.conf \
 && grep -qx 'pid /tmp/nginx.pid;' /etc/nginx/nginx.conf \
 && mkdir -p /var/cache/nginx/client_temp /var/cache/nginx/proxy_temp \
      /var/cache/nginx/fastcgi_temp /var/cache/nginx/uwsgi_temp \
      /var/cache/nginx/scgi_temp \
 && chown -R 101:101 /var/cache/nginx

COPY nginx.conf /etc/nginx/conf.d/syscert.conf
COPY --from=build /app/dist /usr/share/nginx/html

# Fail the BUILD on a bad config rather than at container start.
RUN nginx -t

USER 101:101
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget -qO /dev/null http://127.0.0.1:8080/healthz || exit 1

CMD ["nginx", "-g", "daemon off;"]
