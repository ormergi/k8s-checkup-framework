FROM golang:1.17.7-alpine3.15

WORKDIR /usr/src/checkup-launcher

COPY . .
RUN go mod verify

RUN go build -v -o /usr/local/bin/checkup-launcher

ENTRYPOINT checkup-launcher
