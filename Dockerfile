FROM alpine:latest
ENV HTTP_PROXY http://10.61.9.74:3128/
RUN apk --no-cache add ca-certificates

WORKDIR /root
COPY snowball .
COPY snowball.conf .
