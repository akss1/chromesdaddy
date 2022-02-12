FROM golang:alpine3.15 AS build

ARG BUILD_NAME=chromebalancer

WORKDIR /go/$BUILD_NAME

COPY chrome/ chrome/
COPY clientsstore/ clientsstore/
COPY utils/ utils/
COPY go.mod .
COPY go.sum .
COPY main.go .
COPY server.go .

ENV CGO_ENABLED=0

RUN go build -o /go/bin/$BUILD_NAME

FROM chromedp/headless-shell:98.0.4758.82

ARG BUILD_NAME=chromebalancer

COPY --from=build /go/bin/$BUILD_NAME /usr/local/bin/$BUILD_NAME
COPY entrypoint.sh /usr/local/bin/entrypoint.sh

ENV \
    PORT=9222 \
    MAX_CHROMES_NUM=16

HEALTHCHECK --interval=30s --timeout=30s --start-period=10s --retries=3 CMD /usr/bin/curl -sS http://127.0.0.1:$PORT/healthcheck/ || exit 1

ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
