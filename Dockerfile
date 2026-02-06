FROM node:18-alpine AS dashboard-builder
WORKDIR /app/apps/dashboard
COPY apps/dashboard/package.json apps/dashboard/package-lock.json ./
RUN npm ci
COPY apps/dashboard ./
RUN npm run build

FROM golang:1.23-alpine AS builder
ARG TARGETARCH
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
COPY keys ./keys
COPY config ./config
COPY updates ./updates
RUN GOOS=linux GOARCH=${TARGETARCH} go build -o main ./cmd/api

FROM alpine:latest
RUN apk add --no-cache bash
COPY --from=builder /app/main /app/main
COPY --from=dashboard-builder /app/apps/dashboard/dist /app/apps/dashboard/dist
EXPOSE 3000
CMD ["/app/main"]
