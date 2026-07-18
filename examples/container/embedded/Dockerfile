# syscert embedded in nginx — WITH CAVEATS (read before using)
# ------------------------------------------------------------------------------
# TRADE-OFF: two responsibilities in one container. If syscert fails (e.g. an
# ACME error), nginx keeps serving the existing (possibly stale) cert — silently.
# If nginx crashes, the certificate still renews but nothing serves it.
#
# MITIGATION (required): use the entrypoint below, which restarts the container if
# either process exits. This surfaces the failure to Docker's restart policy and
# avoids the "silent stale cert" problem.
#
# Prefer the sidecar pattern (compose.sidecar.yml) for production. Use this
# pattern when a single container is a hard requirement (e.g. a constrained PaaS).
#
# Build:
#   docker build --build-arg SYSCERT_VERSION=v0.3.1 -t myapp:latest .
# ------------------------------------------------------------------------------

FROM nginx:alpine

ARG SYSCERT_VERSION=v0.3.1
ARG TARGETARCH=amd64

# Download and verify the syscert binary.
# In production, pin the digest and verify the checksum.
RUN apk add --no-cache curl && \
    curl -fsSL \
      "https://github.com/tfindley/syscert/releases/download/${SYSCERT_VERSION}/syscert-linux-${TARGETARCH}" \
      -o /usr/local/bin/syscert && \
    chmod 0755 /usr/local/bin/syscert && \
    # Verify checksum (replace with the correct sha256 for the pinned version):
    # echo "<sha256sum>  /usr/local/bin/syscert" | sha256sum -c - && \
    apk del curl

# Non-root: create a syscert user that shares the nginx group.
# nginx runs as uid 101; syscert needs to write /var/lib/syscert (its own store)
# and deliver files nginx can read (group nginx, mode 0640 or 0644).
RUN adduser -D -u 1001 -G nginx syscert && \
    mkdir -p /var/lib/syscert /etc/syscert && \
    chown syscert:nginx /var/lib/syscert && \
    chmod 0750 /var/lib/syscert

COPY syscert.toml /etc/syscert/syscert.toml
COPY entrypoint.sh /entrypoint.sh
RUN chmod 0755 /entrypoint.sh

# Secrets: do NOT bake DNS credentials into the image.
# Pass them at runtime via --env-file or Docker secrets (see entrypoint.sh).

EXPOSE 80 443

ENTRYPOINT ["/entrypoint.sh"]
