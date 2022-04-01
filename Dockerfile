FROM golang:1.17.8-buster AS build
WORKDIR /otel
COPY . .
RUN go get ./...
RUN GO111MODULE=on go install go.opentelemetry.io/collector/cmd/builder@latest
RUN builder --config otelcol-builder.yaml --output-path=./build --name="nt-otelcol"
RUN cd build && CGO_ENABLED=0 go build

FROM golang:1.17.8-buster
COPY --from=build /otel/build .
ENTRYPOINT ["./nt-otelcol", "--config=/conf/otelcol.yaml"]
