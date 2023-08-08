# Simple URL Shortener

[![CI](https://github.com/vancanhuit/url-shortener-api/actions/workflows/ci.yml/badge.svg)](https://github.com/vancanhuit/url-shortener-api/actions/workflows/ci.yml)

Install the following tools:
* [Golang](https://go.dev/dl/)
* [Docker Engine](https://docs.docker.com/engine/install/)
* [Docker Compose](https://docs.docker.com/compose/)

Running test: `go test -v -cover ./...`

Running API locally:
```sh
$ docker compose up -d --build
# NOTE: The generated alias will be different in every run
$ curl -X POST \
        -H "Content-Type: application/json" \
        http://localhost:9000/api/shorten -d '{"url": "https://reddit.com"}'
HTTP/1.1 201 Created
Content-Type: application/json
Date: Tue, 08 Aug 2023 12:54:54 GMT
Content-Length: 69

{"data":{"original_url":"https://reddit.com","alias":"sPQAW_f7HdB"}}

$ curl -i http://localhost:9000/sPQAW_f7HdB
HTTP/1.1 302 Found
Content-Type: text/html; charset=utf-8
Location: https://reddit.com
Date: Tue, 08 Aug 2023 12:55:19 GMT
Content-Length: 41

<a href="https://reddit.com">Found</a>.

$ curl -i -X DELETE http://localhost:9000/sPQAW_f7HdB
HTTP/1.1 204 No Content
Date: Tue, 08 Aug 2023 12:55:49 GMT
```
