FROM goreleaser/goreleaser:latest as builder

WORKDIR /build
ADD . /build

RUN goreleaser build

FROM alpine:latest
LABEL org.opencontainers.image.source=https://github.com/cirruslabs/cirrus-cli/

COPY --from=builder /build/dist/cirrus_linux_amd64/cirrus /usr/local/bin/
ENTRYPOINT ["/usr/local/bin/cirrus"]
