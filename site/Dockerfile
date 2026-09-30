# Build the static site, then serve it with nginx.
#
# Built from the REPOSITORY ROOT, not from site/: `docker build -f
# site/Dockerfile .`, and in Coolify Base Directory `/` with Dockerfile
# Location `/site/Dockerfile`. The root is the context so that the image
# can read the commit from .git and the version from VERSION itself, and
# not depend on a build argument the deployer may or may not pass. The
# root .dockerignore keeps the context to site/, .git and VERSION.

# Stage 1: Build
FROM node:22-slim AS builder

RUN corepack enable

WORKDIR /app
COPY site/package.json site/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile

COPY site/ .

# Drop nitro's build manifest before it can be copied anywhere. `nitro build`
# writes dist/static/nitro.json -- a build timestamp, the nitro version and an
# `npx serve .` preview command -- and the COPY in stage 2 would otherwise take
# it with everything else. Nothing on the site fetches it, so serving it only
# adds a public URL that changes on every deploy.
#
# Removed rather than given a cache rule in nginx.conf: a file nobody requests
# does not need a correct cache policy, it needs to not be there. The
# /version.json rule in nginx.conf exists because that file IS fetched, by the
# deploy gate; this one has no such reader.
#
# In this stage, not after the COPY. `COPY` then `RUN rm` in the serving stage
# only writes a whiteout: the 232 bytes stay in the layer underneath and ship
# with every pull. Deleting it here means the file never reaches a layer that
# is kept. Confirmed with `docker save`, see the pull request.
#
# nginx then answers /nitro.json with a 404, and the `location /` block
# deliberately omits `always`, so that 404 carries no Cache-Control at all.
RUN pnpm build && rm -f dist/static/nitro.json

# Stage 2: Serve
FROM nginx:alpine

COPY --from=builder /app/dist/static /usr/share/nginx/html
COPY site/nginx.conf /etc/nginx/conf.d/default.conf

# Which commit, and which version, is actually being served.
#
# A static site has no /api/health to ask, so bake the answer in. Without it a
# post-deploy probe cannot tell the new container from the old one: it passes
# against whatever is still running and a failed deploy looks green.
#
# An explicit --build-arg GIT_SHA wins (CI passes it). Otherwise the commit
# is read from the checkout's own .git: HEAD names a ref, the ref is a loose
# file or a line in packed-refs. No git in this image, so a shell does it.
# Coolify passed neither GIT_SHA nor SOURCE_COMMIT, which is why the site
# once said "unknown".
ARG GIT_SHA
COPY .git /tmp/git
COPY VERSION /tmp/VERSION
RUN set -e; \
    sha="$GIT_SHA"; \
    if [ -z "$sha" ] && [ -f /tmp/git/HEAD ]; then \
      head="$(cat /tmp/git/HEAD)"; \
      case "$head" in \
        ref:*) ref="${head#ref: }"; \
               if [ -f "/tmp/git/$ref" ]; then sha="$(cat "/tmp/git/$ref")"; \
               elif [ -f /tmp/git/packed-refs ]; then sha="$(grep " $ref\$" /tmp/git/packed-refs | cut -c1-40)"; fi ;; \
        *) sha="$head" ;; \
      esac; \
    fi; \
    printf '{"commit":"%s","version":"%s"}\n' "${sha:-unknown}" "$(tr -d '[:space:]' < /tmp/VERSION)" \
      > /usr/share/nginx/html/version.json; \
    rm -rf /tmp/git /tmp/VERSION

EXPOSE 80

# Coolify (and any orchestrator) uses this to decide whether a deploy actually
# came up, rather than assuming a started container is a working one. Requests
# a real page, so a container that boots with an empty document root fails
# instead of reporting healthy.
# 127.0.0.1, not localhost: inside the container "localhost" resolves to ::1
# first, and a check against an IPv6 address fails with "connection refused"
# unless nginx is also listening there -- which would report every healthy
# deploy as unhealthy.
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --quiet --tries=1 --spider http://127.0.0.1/ || exit 1

CMD ["nginx", "-g", "daemon off;"]
