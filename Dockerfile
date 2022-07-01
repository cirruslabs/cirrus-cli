FROM goreleaser/goreleaser:latest as builder

WORKDIR /build
ADD . /build

RUN goreleaser build --timeout 60m --single-target

FROM alpine:latest
LABEL org.opencontainers.image.source=https://github.com/cirruslabs/cirrus-cli/

COPY --from=builder /build/dist/cirrus_linux_*/cirrus /usr/local/bin/
ENTRYPOINT ["/usr/local/bin/cirrus"]
