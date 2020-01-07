package server

import (
	"net/http"
	"strings"

	"github.com/ysmood/kit"
)

type proxy struct {
	host   string
	status map[string]interface{}
}

type itemType int

const (
	itemRead = iota
	itemSend
	itemReq
	itemWSReq
)

type proxyItem struct {
	id        string
	itemType  itemType
	subdomain string
	ctx       kit.GinContext
}

func newProxy(host string) *proxy {
	return &proxy{
		host: host,
	}
}

func (p *proxy) handler(ctx kit.GinContext) {
	var itemType itemType
	var subdomain string

	if ctx.Request.Host == p.host {
		subdomain = strings.Trim(ctx.Request.URL.Path, "/")
		if ctx.Request.Method == http.MethodGet {
			itemType = itemRead
		} else {
			itemType = itemSend
		}
	} else {
		subdomain = strings.Replace(ctx.Request.Host, "."+p.host, "", 1)
		if isWebsocket(ctx) {
			itemType = itemWSReq
		} else {
			itemType = itemReq
		}
	}

	id := ctx.GetHeader("Digto-ID")
	if id == "" {
		id = randString()
	}

	_ = proxyItem{
		id:        id,
		itemType:  itemType,
		subdomain: subdomain,
		ctx:       ctx,
	}
}

func isWebsocket(ctx kit.GinContext) bool {
	return ctx.GetHeader("Upgrade") == "websocket"
}
