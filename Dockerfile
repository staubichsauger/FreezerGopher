# STAGE 1: Go build
FROM golang:buster AS build

RUN apt-get update \
 && apt-get install build-essential sqlite3 -y

WORKDIR /gopherize
COPY . .

RUN go test ./... \
 && go vet -v ./...
RUN go build -o gopherize.bin .


# STAGE 3: Final image
FROM scratch

WORKDIR /gopherize
COPY --from=build /gopherize/gopherize.bin .

EXPOSE 8181

RUN chown -R 1000:1000 /gopherize

USER 1000
ENTRYPOINT "./gopherize.bin"
