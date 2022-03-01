FROM alpine:3.15.0

RUN apk add --no-cache curl

RUN curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl" &&\
    chmod 744 kubectl &&\
    mv ./kubectl /usr/local/bin

COPY ./entrypoint /usr/local/bin

ENTRYPOINT entrypoint
