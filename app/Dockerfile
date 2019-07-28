FROM golang:1.12.5-alpine3.9

RUN apk update && \
    apk add --no-cache \
    openssl \
    git

ENV DOCKERIZE_VERSION v0.6.1
RUN wget https://github.com/jwilder/dockerize/releases/download/$DOCKERIZE_VERSION/dockerize-alpine-linux-amd64-$DOCKERIZE_VERSION.tar.gz \
    && tar -C /usr/local/bin -xzvf dockerize-alpine-linux-amd64-$DOCKERIZE_VERSION.tar.gz \
    && rm dockerize-alpine-linux-amd64-$DOCKERIZE_VERSION.tar.gz

COPY . /app
WORKDIR /app

RUN go get -d -v ./...
RUN go build main.go

EXPOSE ${APP_PORT}

ENTRYPOINT ["./main"]