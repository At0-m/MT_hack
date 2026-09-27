FROM golang:1.26.8-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -tags onnx -trimpath -o /out/api ./cmd/api && \
    CGO_ENABLED=1 go build -tags onnx -trimpath -o /out/publisher ./cmd/publisher && \
    go build -trimpath -o /out/fixtures ./cmd/fixtures && \
    go build -trimpath -o /out/loadtest ./cmd/loadtest

FROM debian:bookworm-slim AS loadtest
WORKDIR /app
COPY --from=build /out/loadtest /app/loadtest
ENTRYPOINT ["/app/loadtest"]

FROM debian:bookworm-slim AS runtime
ARG ORT_VERSION=1.24.1
ARG ORT_SHA256=9142552248b735920f9390027e4512a2cacf8946a1ffcbe9071a5c210531026f
RUN apt-get update \
    && apt-get install -y --no-install-recommends \
         ca-certificates \
         curl \
         libgomp1 \
    && curl --fail --location --retry 3 \
         "https://github.com/microsoft/onnxruntime/releases/download/v${ORT_VERSION}/onnxruntime-linux-x64-${ORT_VERSION}.tgz" \
         -o /tmp/ort.tgz \
    && echo "${ORT_SHA256}  /tmp/ort.tgz" | sha256sum --check --strict \
    && tar -xzf /tmp/ort.tgz -C /opt \
    && rm /tmp/ort.tgz \
    && rm -rf /var/lib/apt/lists/* \
    && useradd --system --uid 10001 --create-home tramflow
ENV ONNX_LIBRARY_PATH=/opt/onnxruntime-linux-x64-1.24.1/lib/libonnxruntime.so.1.24.1
ENV HTTP_ADDR=0.0.0.0:8080 ARTIFACTS_ROOT=/data/artifacts OPENAPI_PATH=/app/openapi/openapi.yaml
WORKDIR /app
COPY --from=build /out/ /app/
COPY openapi/ /app/openapi/
USER tramflow

EXPOSE 8080 9090

HEALTHCHECK \
  --interval=3s \
  --timeout=3s \
  --start-period=5s \
  --retries=40 \
  CMD curl -fsS http://127.0.0.1:8080/health/ready >/dev/null || exit 1

ENTRYPOINT ["/app/api"]

FROM build AS integration
ENV ONNX_LIBRARY_PATH=/opt/onnxruntime-linux-x64-1.24.1/lib/libonnxruntime.so.1.24.1
COPY --from=runtime /opt/ /opt/
CMD ["sh", "-c", "go run ./cmd/fixtures --onnx && go test -tags onnx -race -count=1 ./..."]

FROM runtime AS demo

# Root нужен только на этапе сборки demo image.
# Сам runtime ниже снова переключается на tramflow.
USER root

RUN apt-get update \
    && apt-get install -y --no-install-recommends postgresql-client \
    && rm -rf /var/lib/apt/lists/*

# fixtures создаёт ./artifacts относительно WORKDIR=/app.
# Во время build разрешаем ему создать эти файлы.
RUN mkdir -p /data /app/artifacts /app/testdata \
    && /app/fixtures \
         --onnx \
         --stress \
         --output /data/demo-bundle.json \
    && mkdir -p /data/artifacts \
    && cp -a /app/artifacts/. /data/artifacts/ \
    && rm -rf /app/artifacts /app/testdata \
    && chown -R tramflow:tramflow /data

COPY --chown=tramflow:tramflow scripts/demo-init.sh /app/demo-init

RUN chmod 0555 /app/demo-init

# Runtime обратно non-root.
USER tramflow

WORKDIR /app

ENTRYPOINT ["/app/api"]