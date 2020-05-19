################################################################
#### base image
FROM golang:1.14-buster as base

RUN apt-get update && apt-get -y install git unzip build-essential autoconf libtool

RUN git clone https://github.com/google/protobuf.git /protobuf
RUN cd /protobuf &&\
    ./autogen.sh &&\
    ./configure &&\
    make &&\
    make install &&\
    ldconfig &&\
    make clean &&\
    cd .. &&\
    rm -r protobuf

RUN go get google.golang.org/grpc
RUN go get github.com/golang/protobuf/protoc-gen-go

################################################################
#### pkgs container (to speed up rebuild after context change; 2x in this case)
FROM base as pkgs

WORKDIR /src

COPY ./go.mod ./go.sum ./

RUN go mod download -x

################################################################
#### intermediate container
FROM base as prepared

COPY --from=pkgs $GOPATH/ $GOPATH/

WORKDIR /src
COPY . .

################################################################
#### main container
FROM prepared as main

WORKDIR /src

RUN go generate
RUN go build -o /grpc-example

WORKDIR /

################################################################
#### test server container
FROM prepared as test_server

WORKDIR /src/test-server

RUN go build -o /test-server

WORKDIR /
