# syntax=docker/dockerfile:1.7

# ---- frontend ---------------------------------------------------------------
FROM node:22-alpine AS frontend
WORKDIR /src
COPY src/frontend/package.json src/frontend/package-lock.json* ./
RUN npm install
COPY src/frontend ./
RUN npm run build
# Output lands in /src/dist

# ---- backend ----------------------------------------------------------------
FROM golang:1.23-alpine AS backend
WORKDIR /src/backend

# Cache modules
COPY src/backend/go.mod src/backend/go.sum* ./
RUN go mod download

COPY src/backend/ ./
# Bring the built SPA into the backend source tree so embed.FS picks it up
COPY --from=frontend /src/dist ./web/dist

# Static binary, trimmed
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -trimpath \
    -o /out/quintus \
    ./cmd/quintus

# ---- final ------------------------------------------------------------------
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=backend /out/quintus /quintus
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/quintus"]
