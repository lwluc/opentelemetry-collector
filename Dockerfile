FROM golang:1.17 AS build

WORKDIR /otel

COPY . .

RUN go get ./...
RUN GO111MODULE=on go install go.opentelemetry.io/collector/cmd/builder@latest
RUN builder --config otelcol-builder.yaml --output-path=./build --name="nt-otelcol"
RUN cd build && export CGO_ENABLED=0 && go build

FROM alpine
RUN apk add --no-cache ca-certificates
COPY --from=build /otel/build .
RUN chmod +x ./nt-otelcol
ENTRYPOINT ["./nt-otelcol"]