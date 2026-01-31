FROM golang:latest AS builder

WORKDIR /build
ADD . /build

# Install GoReleaser Pro
RUN echo 'deb [trusted=yes] https://repo.goreleaser.com/apt/ /' | tee /etc/apt/sources.list.d/goreleaser.list
RUN apt update && apt -y install goreleaser-pro

RUN goreleaser build --timeout 60m --single-target

# Temporarily downgraded due to "execve: No such file or directory" error [1].
#
# [1]: https://gitlab.alpinelinux.org/alpine/aports/-/issues/17775
FROM alpine:3.22
LABEL org.opencontainers.image.source=https://github.com/cirruslabs/cirrus-cli/
RUN apk add --no-cache rsync
COPY --from=builder /build/dist/linux_*/cirrus_linux_*/cirrus /usr/local/bin/
ENTRYPOINT ["/usr/local/bin/cirrus"]
