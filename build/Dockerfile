FROM alpine:3.15.0

RUN apk add --no-cache libc6-compat

COPY checkup-framework /usr/local/bin

ENTRYPOINT checkup-framework
