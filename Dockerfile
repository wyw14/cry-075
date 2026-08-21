FROM node:22-alpine AS web
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM golang:1.24-alpine AS go-builder
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build CGO_ENABLED=0 go build -o /out/server ./cmd/server

FROM alpine:3.21
RUN addgroup -S app && adduser -S -G app app
WORKDIR /app
COPY --from=go-builder /out/server /app/server
COPY --from=web /src/web/dist /app/web/dist
USER app
EXPOSE 8080
ENTRYPOINT ["/app/server"]
