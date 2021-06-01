FROM golang:1.16 as build-server

WORKDIR /go/src/github.com/ssb-ngi-pointer/go-ssb-room
COPY . .

RUN cd cmd/server && go build -trimpath -v .

FROM gcr.io/distroless/base

COPY --from=build-server /go/src/github.com/ssb-ngi-pointer/go-ssb-room/cmd/server/server /server

EXPOSE 8008
EXPOSE 3000

CMD ["/server"]