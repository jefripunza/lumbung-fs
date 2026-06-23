FROM oven/bun:latest AS frontend
FROM golang:1.25-bookworm AS backend
FROM debian:bookworm-slim AS runner
LABEL org.opencontainers.image.authors="Jefri Herdi Triyanto <jefriherditriyanto@gmail.com>"
LABEL description="LumbungFS is a premium, developer-focused, self-hosted object storage and file platform."


# =======================================================================================
# Build Frontend
# =======================================================================================

FROM frontend AS fe-builder
WORKDIR /app

# install dependencies
COPY ./web/package.json ./
RUN bun install

# build
COPY ./web/ .
ENV VITE_IS_DOCKER=true
RUN bun run build-only

# =======================================================================================
# Build Backend
# =======================================================================================

FROM backend AS be-builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
COPY --from=fe-builder /app/dist ./web/dist

RUN go build -o lumbung-fs main.go

# =======================================================================================
FROM runner
WORKDIR /app

RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*

# copy compiled files
COPY --from=be-builder /app/lumbung-fs /app/lumbung-fs
COPY --from=be-builder /app/web/dist /app/web/dist

# run
CMD ["./lumbung-fs"]
