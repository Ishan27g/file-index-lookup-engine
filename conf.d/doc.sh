#!/bin/sh

# from root `./conf.d/doc.sh fsi-raft-service`
version=v0.2

cp ./conf.d/Dockerfile .
docker build -t "$1":$version .
docker tag "$1":$version ishan27g/"$1":$version
rm Dockerfile

# docker push ishan27g/fsi-raft-service:v0.2