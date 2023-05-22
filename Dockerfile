# Base image
FROM golang:1.20-buster

RUN mkdir /app/sandbox

WORKDIR /app

EXPOSE 8080