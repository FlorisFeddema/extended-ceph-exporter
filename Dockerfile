FROM golang:1.26-bookworm AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal

ARG VERSION=dev
ARG TARGETOS=linux
ARG TARGETARCH

RUN target_arch="${TARGETARCH:-$(go env GOARCH)}" && \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${target_arch} \
    go build -trimpath -ldflags="-s -w" \
    -o /out/extended-ceph-exporter ./cmd/extended-ceph-exporter

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /out/extended-ceph-exporter /extended-ceph-exporter

EXPOSE 9877

ENTRYPOINT ["/extended-ceph-exporter"]
