FROM golang:latest as builder

WORKDIR /build
ADD . /build

# Install GoReleaser Pro
RUN echo 'deb [trusted=yes] https://repo.goreleaser.com/apt/ /' | tee /etc/apt/sources.list.d/goreleaser.list
RUN apt update && apt -y install goreleaser-pro

RUN goreleaser build --timeout 60m --single-target

FROM alpine:latest
LABEL org.opencontainers.image.source=https://github.com/cirruslabs/cirrus-cli/

COPY --from=builder /build/dist/linux_*/cirrus_linux_*/cirrus /usr/local/bin/
ENTRYPOINT ["/usr/local/bin/cirrus"]
