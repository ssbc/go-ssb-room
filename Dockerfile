FROM golang:1.16-alpine

RUN apk add --no-cache \
      build-base \
      git \
      sqlite \
      sqlite-dev

RUN mkdir /app
WORKDIR /app
COPY . /app

RUN cd /app/cmd/server && go build && \
    cd /app/cmd/insert-user && go build

EXPOSE 8008
EXPOSE 3000

RUN apk del \
      build-base \
      git

CMD ./start.sh
