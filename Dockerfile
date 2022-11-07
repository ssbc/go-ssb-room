# SPDX-FileCopyrightText: 2021 The NGI Pointer Secure-Scuttlebutt Team of 2020/2021
#
# SPDX-License-Identifier: Unlicense

FROM golang:1.17-alpine

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
