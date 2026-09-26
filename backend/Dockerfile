FROM golang:1.26.2-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -tags onnx -trimpath -o /out/api ./cmd/api && \
    CGO_ENABLED=1 go build -tags onnx -trimpath -o /out/publisher ./cmd/publisher && \
    go build -o /out/fixtures ./cmd/fixtures

FROM debian:bookworm-slim AS runtime
ARG ORT_VERSION=1.24.1
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates curl libgomp1 && \
    curl --fail --location --retry 3 "https://github.com/microsoft/onnxruntime/releases/download/v${ORT_VERSION}/onnxruntime-linux-x64-${ORT_VERSION}.tgz" -o /tmp/ort.tgz && \
    tar -xzf /tmp/ort.tgz -C /opt && rm /tmp/ort.tgz && \
    apt-get purge -y curl && apt-get autoremove -y && rm -rf /var/lib/apt/lists/* && \
    useradd --system --uid 10001 --create-home tramflow
ENV ONNX_LIBRARY_PATH=/opt/onnxruntime-linux-x64-1.24.1/lib/libonnxruntime.so.1.24.1
ENV HTTP_ADDR=0.0.0.0:8080 ARTIFACTS_ROOT=/data/artifacts OPENAPI_PATH=/app/openapi/openapi.yaml
WORKDIR /app
COPY --from=build /out/ /app/
COPY openapi/ /app/openapi/
USER tramflow
EXPOSE 8080
ENTRYPOINT ["/app/api"]

FROM build AS integration
ENV ONNX_LIBRARY_PATH=/opt/onnxruntime-linux-x64-1.24.1/lib/libonnxruntime.so.1.24.1
COPY --from=runtime /opt/ /opt/
CMD ["sh", "-c", "go run ./cmd/fixtures --onnx && go test -tags onnx -race -count=1 ./..."]
