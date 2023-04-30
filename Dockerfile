FROM golang:1.19-buster as builder

WORKDIR /app
COPY go.* ./
RUN go mod download

COPY . ./
RUN go build -v -o server

FROM debian:buster-slim
RUN set -x && apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y \
    ca-certificates && \
    rm -rf /var/lib/apt/lists/*

RUN apt update && apt install libxrender1 libfontconfig1 libxext6 -y

ENV SERVER_PORT 8888
ENV SERVER_HOST 0.0.0.0

COPY --from=builder /app/wkhtmltopdf /usr/local/bin/wkhtmltopdf
COPY --from=builder /app/templates/. /app/templates/
COPY --from=builder /app/server /app/server

WORKDIR /app
EXPOSE 8888

CMD ["./server"]