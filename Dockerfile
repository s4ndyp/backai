# syntax=docker/dockerfile:1

FROM node:22-bookworm AS web
WORKDIR /src/web
COPY web/package.json web/package-lock.json* ./
RUN npm ci || npm install
COPY web/ ./
RUN npm run build

FROM golang:1.22-bookworm AS server
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web /src/cmd/server/static ./cmd/server/static
ARG VERSION=dev
ARG COMMIT=none
ARG BUILDTIME=unknown
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-s -w \
  -X github.com/s4ndyp/backai/internal/version.Version=${VERSION} \
  -X github.com/s4ndyp/backai/internal/version.Commit=${COMMIT} \
  -X github.com/s4ndyp/backai/internal/version.BuildTime=${BUILDTIME}" \
  -o /out/backai-server ./cmd/server

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=server /out/backai-server /app/backai-server
ENV BACKAI_HTTP_ADDR=:8080
EXPOSE 8080
USER nobody
ENTRYPOINT ["/app/backai-server"]
