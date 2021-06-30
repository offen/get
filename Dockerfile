# Copyright 2020 - Offen Authors <hioffen@posteo.de>
# SPDX-License-Identifier: MPL-2.0

FROM golang:1.16-alpine as builder

WORKDIR /code
COPY go.mod go.sum /code/
RUN go mod download
COPY main.go /code/
RUN go build -o get main.go

FROM alpine:3.14

RUN addgroup -g 10001 -S get && \
  adduser -u 10000 -S -G get -h /home/get get
RUN apk add -U --no-cache ca-certificates tini libcap

COPY --from=builder /code/get /sbin/get
RUN setcap CAP_NET_BIND_SERVICE=+eip /sbin/get

ENV PORT 80
EXPOSE 80

ENTRYPOINT ["/sbin/tini", "--", "get"]

USER get
