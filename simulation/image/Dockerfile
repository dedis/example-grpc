# https://github.com/grpc/grpc-docker-library/blob/master/1.0/golang/Dockerfile

FROM golang:1.13

RUN apt-get update && apt-get -y install unzip && apt-get clean

# install protobuf
ENV PB_VER 3.6.0
ENV PB_URL https://github.com/google/protobuf/releases/download/v${PB_VER}/protoc-${PB_VER}-linux-x86_64.zip
RUN mkdir -p /tmp/protoc && \
    curl -L ${PB_URL} > /tmp/protoc/protoc.zip && \
    cd /tmp/protoc && \
    unzip protoc.zip && \
    cp /tmp/protoc/bin/protoc /usr/local/bin && \
    cp -R /tmp/protoc/include/* /usr/local/include && \
    chmod go+rx /usr/local/bin/protoc && \
    cd /tmp && \
    rm -r /tmp/protoc

RUN go get -u github.com/golang/protobuf/protoc-gen-go

RUN go get -d github.com/dedis/example-grpc/overlay
RUN cd $GOPATH/src/github.com/dedis/example-grpc && \
    make generate

RUN mkdir /home/conode
ADD main.go /home/conode/main.go

RUN cd /home/conode && \
    go build main.go

EXPOSE 2000

CMD ["/home/conode/main", "2000"]
