FROM alpine:3.15.0

RUN apk add --no-cache libc6-compat

COPY kubevirt-latency /usr/local/bin

ENTRYPOINT kubevirt-latency
