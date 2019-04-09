This repository implements simple jaeger client-server flow

It uses jaeger docker for simple environment setup
`docker run -d -p6831:6831/udp -p16686:16686 jaegertracing/all-in-one:latest`

Usage:

1. Run service

    go run service/receiver.go

2. Run client

    go run client/client.go
