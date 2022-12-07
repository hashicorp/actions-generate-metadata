# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

FROM golang:1.18 as build
MAINTAINER Team Rel Eng team-rel-eng@hashicorp.com

# Copy all the action files into the container
WORKDIR /go/src/action
COPY action /go/src/action

# Enable Go modules
ENV GO111MODULE=on
RUN go get -d -v

# Compile the action
RUN CGO_ENABLED=0 go build -o /action -ldflags="-s -w" action.go

FROM alpine:latest
RUN apk --update add ca-certificates
RUN apk add --no-cache git make bash

COPY --from=build /action /
# Specify the container's entrypoint as the action
ENTRYPOINT ["/action"]
