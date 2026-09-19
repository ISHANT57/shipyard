# syntax=docker/dockerfile:1

# --- Build stage: has the Go toolchain, produces one static binary ---
FROM golang:1.24-bookworm AS builder
WORKDIR /src

# Copy module files first and download deps before the rest of the
# source — Docker caches this layer, so `go mod download` only re-runs
# when go.mod/go.sum actually change, not on every source edit.
COPY go.mod ./
RUN go mod download

COPY . .

# CGO_ENABLED=0 produces a statically linked binary with no libc
# dependency, which is what lets the final stage be distroless (no shell,
# no package manager, nothing beyond the binary itself and its runtime
# dependencies) rather than needing a full OS base image.
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/shipyard-api ./cmd/shipyard-api

# --- Final stage: no shell, no package manager, runs as a fixed non-root UID ---
FROM gcr.io/distroless/static-debian12:nonroot AS final

COPY --from=builder /out/shipyard-api /shipyard-api

# The "nonroot" distroless variant already runs as UID 65532; this is
# explicit anyway so the intent is visible without knowing that fact.
USER 65532:65532

EXPOSE 8080
ENTRYPOINT ["/shipyard-api"]
