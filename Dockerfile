# syntax=docker/dockerfile:1

ARG POCKETBASE_VERSION=0.40.4

FROM node:22-bookworm AS web
WORKDIR /src/web
COPY web/package.json web/package-lock.json* ./
RUN npm ci || npm install
COPY web/ ./
RUN npm run build

FROM alpine:3.21
ARG POCKETBASE_VERSION
ARG VERSION=dev
ARG COMMIT=none
ARG BUILDTIME=unknown

ENV BACKAI_VERSION=${VERSION} \
    BACKAI_COMMIT=${COMMIT} \
    BACKAI_BUILD_TIME=${BUILDTIME}

RUN apk add --no-cache ca-certificates curl unzip tini \
    && addgroup -S pocketbase \
    && adduser -S pocketbase -G pocketbase

WORKDIR /app

RUN curl -fsSL \
    "https://github.com/pocketbase/pocketbase/releases/download/v${POCKETBASE_VERSION}/pocketbase_${POCKETBASE_VERSION}_linux_amd64.zip" \
    -o /tmp/pocketbase.zip \
    && unzip /tmp/pocketbase.zip -d /usr/local/bin/ \
    && chmod +x /usr/local/bin/pocketbase \
    && rm /tmp/pocketbase.zip

COPY pb_migrations/ ./pb_migrations/
COPY pb_hooks/ ./pb_hooks/
COPY --from=web /src/pb_public/ ./pb_public/
COPY docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh

RUN chmod +x /usr/local/bin/docker-entrypoint.sh \
    && mkdir -p pb_data \
    && chown -R pocketbase:pocketbase /app

USER pocketbase

EXPOSE 8090

VOLUME ["/app/pb_data"]

ENTRYPOINT ["tini", "--", "/usr/local/bin/docker-entrypoint.sh"]
