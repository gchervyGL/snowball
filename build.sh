#!/bin/sh
echo Building iandri/snowball:build

docker build --build-arg https_proxy=$https_proxy --build-arg http_proxy=$http_proxy \
    -t iandri/snowball:build . -f Dockerfile.build

docker create --name extract iandri/snowball:build
docker cp extract:/go/src/github.com/iandri/snowball/snowball ./snowball
docker rm -f extract

echo Building iandri/snowball:latest

docker build --no-cache -t iandri/snowball:latest .