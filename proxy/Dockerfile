FROM golang:1.27.3-bookworm@sha256:d9b6a1beb9474adc2a1ce842573639b44b2a0dd49b6669f51d3821baf9ec37c9 AS build

WORKDIR /src
COPY proxy/go.mod ./
COPY proxy/main.go proxy/main_test.go ./
RUN go test -race ./...
RUN go vet ./...
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/sub2api-fast-toggle .

FROM scratch
LABEL org.opencontainers.image.title="Sub2API Fast Toggle"
LABEL org.opencontainers.image.description="Local Fast-mode switch for Sub2API Responses API traffic"
LABEL org.opencontainers.image.licenses="MIT"
LABEL org.opencontainers.image.version="0.1.0"
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /usr/share/doc/ca-certificates/copyright /licenses/ca-certificates-copyright.txt
COPY --from=build /out/sub2api-fast-toggle /sub2api-fast-toggle
COPY LICENSE NOTICE.md THIRD_PARTY_NOTICES.md LICENSES/Go-BSD-3-Clause.txt /licenses/
USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/sub2api-fast-toggle"]
