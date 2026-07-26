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

FROM docker.io/library/debian:bookworm-slim
RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates poppler-utils \
    && rm -rf /var/lib/apt/lists/* \
    && useradd --system --uid 65532 --no-create-home --shell /usr/sbin/nologin nonroot
COPY --from=build /out/worker /worker
USER nonroot
ENTRYPOINT ["/worker"]
