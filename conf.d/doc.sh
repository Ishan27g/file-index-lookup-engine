# shellcheck disable=SC2164
cp ./conf.d/Dockerfile .
docker build -t "$1" .
docker tag $1 $1
rm Dockerfile