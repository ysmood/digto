# Digto

[![GoDoc](https://godoc.org/github.com/ysmood/digto?status.svg)](https://godoc.org/github.com/ysmood/digto)
[![codecov](https://codecov.io/gh/ysmood/digto/branch/master/graph/badge.svg)](https://codecov.io/gh/ysmood/digto)
[![goreport](https://goreportcard.com/badge/github.com/ysmood/digto)](https://goreportcard.com/report/github.com/ysmood/digto)

A service to help to expose HTTP/HTTPS service to the public network.
The interface is designed to be easily programmable.
For example you can use `curl` command only to serve public https request without any other dependency.

## Proxy a local port

1. Install the client: `curl -L https://git.io/fjaxx | repo=ysmood/digto sh`

1. Run `digto my-domain :8080` to proxy `https://my-domain.digto.org` to port 8080

### Use `curl` only to handle a request

Open a terminal to send the request:

```bash
curl https://my-subdomain.digto.org/path -d 'ping'
# pong
```

`my-subdomain` can be anything you want. As you can see the request will hang until we send a response back.
Let's open a new terminal to send the response for it:

```bash
curl -i https://digto.org/my-subdomain
# HTTP/2 200
# digto-method: GET
# digto-url: /path
# digto-id: 3dd4e560
#
# ping

# the value of digto-id header must be the same as the previous one
curl https://digto.org/my-subdomain -H 'digto-id: 3dd4e560' -d 'pong'
```

After we send the response the previous terminal will print `pong`.

### Go

```go
package main

import (
    "bytes"
    "fmt"
    "github.com/ysmood/digto/client"
    "io/ioutil"
)

func main() {
    c := client.New("my-subdomain")

    req, res, _ := c.Next()

    data, _ := ioutil.ReadAll(req.Body)
    fmt.Println(string(data)) // output "my-data"

    res(200, nil, bytes.NewBufferString("it works"))

    // curl https://my-subdomain.digto.org -d my-data
    // output "it works"
}
```

### Node.js

```js
const digto = require('digto')

;(async() => {
    const c = digto({ subdomain: 'my-subdomain' })

    const [res, send] = await c.next()

    console.log(res) // # output "my-data"

    await send({ body: 'it works' })

    // curl https://my-subdomain.digto.org -d my-data
    // output "it works"
})()
```

### Ruby

```ruby
require 'digto'

c = Digto::Client.new 'my-subdomain'

s = c.next

puts s.body.to_s # output "my-data"

s.response(200, {}, body: 'it works')

# curl https://my-subdomain.digto.org -d my-data
# output "it works"
```

## API

A OAuth sequence diagram example:

![diagram](doc/digto_sequence_diagram.svg)

The only dependency for a language to implement a client is an HTTP lib.
Usually, the client code can be only a few lines of code. This is nice to become part of an auto-testing.
Such as the integration test of OAuth and payment callbacks.

### GET `/{subdomain}`

Get the request data from the public.

The response is standard http response with 3 extra headers prefixed with `Digto` like:

```text
HTTP/1.1 200 OK
Digto-ID: {id}
Digto-Method: POST
Digto-URL: /callback
Other-Headers: value

<binary body>
```

Digto will proxy the rest headers transparently.

### POST `/{subdomain}`

Send the response data back to the public.

The request should be standard HTTP request with 2 extra headers prefixed with `Digto` like:

```text
POST /test HTTP/1.1
Digto-ID: {id}
Digto-Status: 200
Your-Own-Headers: value

<binary body>
```

The `{id}` is required, you have to send back the `{id}` from the previous response.

### Error

If a protocol-level error happens the response will have the `Digto-Error: reason` header to report the reason.

## Setup private digto server

You can use my [demo server](https://digto.org) for free, you can also setup your own.

Install server: `curl -L https://git.io/fjaxx | repo=ysmood/digto sh`

For help run `digto --help`.

This project helps to handle the boring part of the proxy server, such automatically obtain and renew the https certificate.
So that all you need is to have the permission of the DNS provider and run the server like the example below.

Example to serve `digto serve --dns-config {token} --host test.com`

The server will add two records on your DNS provider, one is like `@.test.com 1.2.3.4`,
the other one with a wildcard like `*.test.com 1.2.3.4`.

For now only [dnspod](https://www.dnspod.com/?lang=en) is supported.
