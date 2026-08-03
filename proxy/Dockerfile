FROM golang:1.26.5-bookworm@sha256:1ecb7edf62a0408027bd5729dfd6b1b8766e578e8df93995b225dfd0944eb651 AS build

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
