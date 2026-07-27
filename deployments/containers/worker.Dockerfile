FROM docker.io/library/golang:1.25.12-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILT_AT=unknown
RUN CGO_ENABLED=0 go build \
    -trimpath \
    -ldflags="-s -w \
      -X github.com/ZekromNguyen/skawld-maintenance/internal/platform/buildinfo.version=${VERSION} \
      -X github.com/ZekromNguyen/skawld-maintenance/internal/platform/buildinfo.commit=${COMMIT} \
      -X github.com/ZekromNguyen/skawld-maintenance/internal/platform/buildinfo.builtAt=${BUILT_AT}" \
    -o /out/worker ./cmd/worker

# The worker binary is statically linked. Alpine supplies only the runtime
# dependencies required for TLS and Poppler-based PDF extraction.
FROM docker.io/library/alpine:3.23
RUN apk add --no-cache ca-certificates poppler-utils \
    && adduser -S -D -H -u 65532 nonroot
COPY --from=build /out/worker /worker
USER nonroot
ENTRYPOINT ["/worker"]
