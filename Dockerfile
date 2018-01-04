ARG GO_VERSION=1.9.2

FROM golang:${GO_VERSION} AS BUILD
LABEL maintainer="frankgreco@northwesternmutual.com"
LABEL version="${VERSION}"
ARG VERSION=""
WORKDIR /go/src/github.com/northwesternmutual/kanali-plugin-apikey/
COPY Gopkg.toml Gopkg.lock Makefile /go/src/github.com/northwesternmutual/kanali-plugin-apikey/
RUN make install
COPY ./ /go/src/github.com/northwesternmutual/kanali-plugin-apikey/
RUN GOOS=`go env GOHOSTOS` GOARCH=`go env GOHOSTARCH` go build -buildmode=plugin -o apiKey_v2.0.0-rc.1.so

FROM alpine:3.7
LABEL maintainer="frankgreco@northwesternmutual.com"
LABEL version="${VERSION}"
COPY --from=BUILD /go/src/github.com/northwesternmutual/kanali-plugin-apikey/apiKey_v2.0.0-rc.1.so /