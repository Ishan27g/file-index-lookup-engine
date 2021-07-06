#!/bin/sh

# to build docker image -> ./conf.d/doc.sh fsi-raft-service

cp ./conf.d/Dockerfile .
docker build -t "$1" .
docker tag $1 $1
rm Dockerfile

# to push new image
# docker tag $1 ishan27g/$1:v0.1