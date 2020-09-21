FROM alpine:latest

COPY cirrus /usr/local/bin/cirrus

ENTRYPOINT ["/usr/local/bin/cirrus"]
