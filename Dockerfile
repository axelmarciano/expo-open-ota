FROM --platform=$BUILDPLATFORM node:24-alpine AS dashboard-builder
WORKDIR /app/apps/dashboard
COPY apps/dashboard/package.json apps/dashboard/package-lock.json ./
RUN npm ci
COPY apps/dashboard ./
RUN npm run build

FROM --platform=$BUILDPLATFORM golang:1.24-alpine AS builder
ARG TARGETARCH
# Stamped into the binary because every cache key embeds version.Version: without
# this the released image reports "development" forever, the keys never rotate,
# and a release that changes a cached payload's shape silently decodes the
# previous release's entries. Release CI passes the git tag; local builds keep
# the package default, which is also what LocalCache.Clear() gates on.
ARG VERSION=development
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
COPY keys ./keys
COPY config ./config
COPY updates ./updates
RUN GOOS=linux GOARCH=${TARGETARCH} go build \
    -ldflags "-X expo-open-ota/internal/version.Version=${VERSION}" \
    -o main ./cmd/api

FROM alpine:latest
RUN apk add --no-cache bash
WORKDIR /app
COPY --from=builder /app/main /app/main
COPY --from=dashboard-builder /app/apps/dashboard/dist /app/apps/dashboard/dist
EXPOSE 3000
CMD ["/app/main"]
