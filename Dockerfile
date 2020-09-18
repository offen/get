# Copyright 2020 - Offen Authors <hioffen@posteo.de>
# SPDX-License-Identifier: MPL-2.0

FROM golang:1.15-alpine as builder

COPY . /code
WORKDIR /code
RUN go build -o get main.go

FROM alpine:3.12

RUN apk add -U --no-cache ca-certificates

COPY --from=builder /code/get /usr/local/bin/get

ENV PORT 80
EXPOSE 80

WORKDIR /root
ENTRYPOINT ["get"]
