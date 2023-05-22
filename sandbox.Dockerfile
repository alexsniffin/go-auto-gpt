# This is a sandbox for running the application in
FROM debian:buster
RUN apt update && \
    apt install -y curl

WORKDIR /tmp
RUN curl https://storage.googleapis.com/golang/go1.16.2.linux-amd64.tar.gz -o go.tar.gz && \
    tar -zxf go.tar.gz && \
    rm -rf go.tar.gz && \
    mv go /go

ENV GOPATH /go
ENV PATH $PATH:/go/bin:$GOPATH/bin

EXPOSE 8080