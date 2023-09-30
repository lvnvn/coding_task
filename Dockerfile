# build
FROM golang:1.21.1-alpine3.18 AS build-env
ADD . /src
RUN cd /src && go build -o goapp

# run
FROM alpine
WORKDIR /app
COPY --from=build-env /src/goapp /app/
EXPOSE 8080
ENTRYPOINT ./goapp
