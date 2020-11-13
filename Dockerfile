# STAGE 1: Go build
FROM golang:buster AS build

RUN apt-get update \
 && apt-get install build-essential sqlite3 -y

WORKDIR /gopheries
COPY . .

RUN go test ./... \
 && go vet -v ./...
RUN go build -o gopheries.bin .


# STAGE 2: Final image
FROM debian:stable-slim

WORKDIR /gopheries/static
ADD static .

WORKDIR /gopheries
COPY --from=build /gopheries/gopheries.bin .

EXPOSE 8181
VOLUME /data

RUN chown -R 1000:1000 /gopheries && chmod +x gopheries.bin

USER 1000
ENTRYPOINT "./gopheries.bin"
