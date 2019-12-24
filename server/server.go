package server

import (
	"crypto/tls"
	"encoding/json"
	"log"
	"net/http"

	"github.com/ysmood/digto/server/cert"
	"github.com/ysmood/kit"
	"github.com/ysmood/storer"
)

// Context ...
type Context struct {
	host      string
	cert      *cert.Context
	server    *kit.ServerContext
	serverTLS *kit.ServerContext
	proxy     *proxy

	onError func(error)
}

// New ...
func New(dbPath, dnsProvider, dnsConfig, host, caDirURL, httpAddr, httpsAddr string) (*Context, error) {
	store := storer.New(dbPath)
	certCache := store.Value("cert-cache", &[]byte{})

	err := setupDNS(dnsProvider, dnsConfig, host)
	if err != nil {
		return nil, err
	}

	cert, err := setupCert(host, dnsProvider, dnsConfig, caDirURL, certCache)
	if err != nil {
		return nil, err
	}

	server, err := kit.Server(httpAddr)
	if err != nil {
		return nil, err
	}

	serverTLS, err := kit.Server(httpsAddr)
	if err != nil {
		return nil, err
	}

	return &Context{
		host:      host,
		cert:      cert,
		server:    server,
		serverTLS: serverTLS,
		proxy:     newProxy(host),
		onError: func(err error) {
			log.Println(err)
		},
	}, nil
}

// GetServer ...
func (ctx *Context) GetServer() *kit.ServerContext {
	return ctx.server
}

// Serve ...
func (ctx *Context) Serve() error {
	ctx.server.Engine.GET("/", ctx.homePage)
	ctx.serverTLS.Engine.GET("/", ctx.homePage)
	ctx.server.Engine.NoRoute(ctx.proxy.handler)
	ctx.serverTLS.Engine.NoRoute(ctx.proxy.handler)

	go ctx.proxy.eventLoop()

	kit.Log(
		"[digto] listen on",
		ctx.server.Listener.Addr().String(),
		ctx.serverTLS.Listener.Addr().String(),
	)

	srv := &http.Server{
		Handler: ctx.serverTLS.Engine,
		TLSConfig: &tls.Config{
			GetCertificate: func(info *tls.ClientHelloInfo) (*tls.Certificate, error) {
				return ctx.cert.Cert(), nil
			},
		},
	}

	go func() {
		kit.Err("[digto]", ctx.server.Do())
	}()

	return srv.ServeTLS(ctx.serverTLS.Listener, "", "")
}

func (ctx *Context) homePage(ginCtx kit.GinContext) {
	if ginCtx.Request.Host != ctx.host || ginCtx.Request.URL.Path != "/" {
		ctx.proxy.handler(ginCtx)
		return
	}

	proxyStatus, _ := json.MarshalIndent(ctx.proxy.status, "", "  ")

	params := []interface{}{
		"version", Version,
		"proxyStatus", string(proxyStatus),
	}

	ginCtx.String(http.StatusOK, kit.S(`
# Digto {{.version}}

## Proxy Status

{{.proxyStatus}}

## API

https://github.com/ysmood/digto	
	`, params...))
}

// ProxyStatus ...
func (ctx *Context) ProxyStatus() map[string]interface{} {
	return ctx.proxy.status
}
