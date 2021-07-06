FROM golang:alpine3.13 as build
WORKDIR /app
COPY . /app/
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o main main.go

FROM alpine:3.13
WORKDIR /app
COPY --from=build /app/main /app/
COPY --from=build /app/.env /app/
COPY --from=build /app/conf.d/entrypoint.sh /app/