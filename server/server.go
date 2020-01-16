package server

import (
	"crypto/tls"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ysmood/digto/server/cert"
	"github.com/ysmood/kit"
	"github.com/ysmood/storer"
)

// Context ...
type Context struct {
	host          string
	cert          *cert.Context
	engine        *gin.Engine
	httpListener  net.Listener
	httpsListener net.Listener
	timeout       time.Duration
	proxy         *proxy

	onError func(error)
}

// New ...
func New(dbPath, dnsProvider, dnsConfig, host, caDirURL, httpAddr, httpsAddr string, timeout time.Duration) (*Context, error) {
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

	httpListener, err := net.Listen("tcp", httpAddr)
	if err != nil {
		return nil, err
	}

	httpsListener, err := net.Listen("tcp", httpsAddr)
	if err != nil {
		return nil, err
	}

	gin.SetMode(gin.ReleaseMode)

	return &Context{
		host:          host,
		cert:          cert,
		engine:        gin.New(),
		httpListener:  httpListener,
		httpsListener: httpsListener,
		timeout:       timeout,
		proxy:         newProxy(host),
		onError: func(err error) {
			log.Println(err)
		},
	}, nil
}

// GetServer ...
func (ctx *Context) GetServer() *kit.ServerContext {
	return &kit.ServerContext{
		Engine:   ctx.engine,
		Listener: ctx.httpListener,
	}
}

// Serve ...
func (ctx *Context) Serve() error {
	ctx.engine.GET("/", ctx.homePage)
	ctx.engine.NoRoute(ctx.proxy.handler)

	go ctx.proxy.eventLoop()

	kit.Log(
		"[digto] listen on",
		ctx.httpListener.Addr().String(),
		ctx.httpsListener.Addr().String(),
	)

	srv := &http.Server{
		Handler:           ctx.engine,
		IdleTimeout:       ctx.timeout,
		ReadHeaderTimeout: ctx.timeout,
		ReadTimeout:       ctx.timeout,
		WriteTimeout:      ctx.timeout,
	}

	go func() {
		kit.Err("[digto]", srv.Serve(ctx.httpListener))
	}()

	tlsSrv := &http.Server{
		Handler:           srv.Handler,
		IdleTimeout:       srv.IdleTimeout,
		ReadHeaderTimeout: srv.ReadHeaderTimeout,
		ReadTimeout:       srv.ReadTimeout,
		WriteTimeout:      srv.WriteTimeout,
		TLSConfig: &tls.Config{
			GetCertificate: func(info *tls.ClientHelloInfo) (*tls.Certificate, error) {
				return ctx.cert.Cert(), nil
			},
		},
	}

	return tlsSrv.ServeTLS(ctx.httpsListener, "", "")
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
