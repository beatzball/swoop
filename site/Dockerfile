# Build the static site, then serve it with nginx.
#
# Built from the REPOSITORY ROOT, not from site/: `docker build -f
# site/Dockerfile .`, and in Coolify Base Directory `/` with Dockerfile
# Location `/site/Dockerfile`. The root is the context so that the image
# can read the commit from .git and the version from VERSION itself, and
# not depend on a build argument the deployer may or may not pass. The
# root .dockerignore keeps the context to site/, .git (when the checkout
# has one) and VERSION.

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

# Which commit, which version, and when it was built.
#
# A static site has no /api/health to ask, so bake the answer in. Without it a
# post-deploy probe cannot tell the new container from the old one: it passes
# against whatever is still running and a failed deploy looks green.
#
# The commit: an explicit --build-arg GIT_SHA wins (CI passes it); else the
# checkout's own .git when the context has one (a clone; HEAD names a ref,
# the ref is a loose file or a line in packed-refs, read in shell since the
# image has no git); else SOURCE_COMMIT, which some deployers pass; else
# "unknown". Coolify imports the repository WITHOUT .git and passes no
# argument, so there the commit is unknown, and the deploy gate in
# .github/workflows/site.yml waits on `built` instead: the seconds since the
# epoch when this layer was made, which a new deploy always moves forward.
#
# VERSION and .git* are one COPY on purpose: a COPY whose every source is
# missing fails the build, and .git may be missing; VERSION never is.
ARG GIT_SHA
ARG SOURCE_COMMIT
COPY VERSION .git* /tmp/src/
RUN set -e; \
    sha="$GIT_SHA"; \
    if [ -z "$sha" ] && [ -f /tmp/src/.git/HEAD ]; then \
      head="$(cat /tmp/src/.git/HEAD)"; \
      case "$head" in \
        ref:*) ref="${head#ref: }"; \
               if [ -f "/tmp/src/.git/$ref" ]; then sha="$(cat "/tmp/src/.git/$ref")"; \
               elif [ -f /tmp/src/.git/packed-refs ]; then sha="$(grep " $ref\$" /tmp/src/.git/packed-refs | cut -c1-40)"; fi ;; \
        *) sha="$head" ;; \
      esac; \
    fi; \
    printf '{"commit":"%s","version":"%s","built":%s}\n' "${sha:-${SOURCE_COMMIT:-unknown}}" "$(tr -d '[:space:]' < /tmp/src/VERSION)" "$(date -u +%s)" \
      > /usr/share/nginx/html/version.json; \
    rm -rf /tmp/src

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
