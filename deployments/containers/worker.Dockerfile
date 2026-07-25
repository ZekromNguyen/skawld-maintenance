FROM golang:1.25.7-bookworm AS build
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

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/worker /worker
ENTRYPOINT ["/worker"]
