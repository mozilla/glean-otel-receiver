FROM alpine:3.19 AS certs
RUN apk --update add ca-certificates

FROM golang:1.24.0 AS build-stage
WORKDIR /build

# Copy Go module files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Install OpenTelemetry Collector builder
RUN --mount=type=cache,target=/root/.cache/go-build GO111MODULE=on go install go.opentelemetry.io/collector/cmd/builder@v0.144.0

# Build custom collector with builder
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    /go/bin/builder --config builder-config.yaml

FROM gcr.io/distroless/base:latest

ARG USER_UID=10001
USER ${USER_UID}

COPY ./collector-config.yaml /etc/otelcol/collector-config.yaml
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --chmod=755 --from=build-stage /build/dist/glean-otelcol /otelcol

ENTRYPOINT ["/otelcol"]
CMD ["--config", "/etc/otelcol/collector-config.yaml"]

EXPOSE 4317 4318 12001
