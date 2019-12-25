package server

import (
	"context"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/ysmood/kit"
)

type proxy struct {
	host         string
	reqConsumers map[string]map[string]*proxyCtx
	resConsumers map[string]*proxyCtx
	reqWaitlist  map[string]map[string]*proxyCtx
	resWaitlist  map[string]*proxyCtx

	consumer      chan *proxyCtx
	reqDone       chan *proxyCtx
	consumerLeave chan *proxyCtx
	req           chan *proxyCtx
	reqLeave      chan *proxyCtx
	res           chan *proxyCtx
	resLeave      chan *proxyCtx

	status map[string]interface{}
}

type proxyCtx struct {
	subdomain string
	id        string
	ctx       kit.GinContext
	cancel    context.CancelFunc
}

func newProxy(host string) *proxy {
	return &proxy{
		host:          host,
		reqConsumers:  map[string]map[string]*proxyCtx{},
		resConsumers:  map[string]*proxyCtx{},
		reqWaitlist:   map[string]map[string]*proxyCtx{},
		resWaitlist:   map[string]*proxyCtx{},
		consumer:      make(chan *proxyCtx),
		reqDone:       make(chan *proxyCtx),
		consumerLeave: make(chan *proxyCtx),
		req:           make(chan *proxyCtx),
		reqLeave:      make(chan *proxyCtx),
		res:           make(chan *proxyCtx),
		resLeave:      make(chan *proxyCtx),
		status:        map[string]interface{}{},
	}
}

func (p *proxy) eventLoop() {
	for {
		select {
		case ctx := <-p.consumer:
			p.add(p.reqConsumers, ctx.subdomain, ctx.id, ctx)
			reqProxyCtx := p.dequeue(p.reqWaitlist, ctx.subdomain)
			if reqProxyCtx != nil {
				p.del(p.reqConsumers, ctx.subdomain, ctx.id)
				ctx.ctx = reqProxyCtx.ctx
				cancel := ctx.cancel
				ctx.cancel = reqProxyCtx.cancel
				cancel()
			}

		case ctx := <-p.req:
			p.add(p.reqWaitlist, ctx.subdomain, ctx.id, ctx)
			consumer := p.dequeue(p.reqConsumers, ctx.subdomain)
			if consumer != nil {
				p.del(p.reqWaitlist, ctx.subdomain, ctx.id)
				consumer.ctx = ctx.ctx
				cancel := consumer.cancel
				consumer.cancel = ctx.cancel
				cancel()
			}

		case ctx := <-p.reqLeave:
			p.del(p.reqWaitlist, ctx.subdomain, ctx.id)

		case ctx := <-p.reqDone:
			p.del(p.reqConsumers, ctx.subdomain, ctx.id)
			p.resConsumers[ctx.id] = ctx

		case ctx := <-p.res:
			p.resWaitlist[ctx.id] = ctx
			consumer, has := p.resConsumers[ctx.id]
			if has {
				delete(p.resWaitlist, ctx.id)
				delete(p.resConsumers, ctx.id)
				consumer.ctx = ctx.ctx
				cancel := consumer.cancel
				consumer.cancel = ctx.cancel
				cancel()
			}

		case ctx := <-p.resLeave:
			delete(p.resWaitlist, ctx.id)

		case ctx := <-p.consumerLeave:
			delete(p.resConsumers, ctx.id)
		}

		p.updateStatus()
	}
}

func (p *proxy) dequeue(dict map[string]map[string]*proxyCtx, subdomain string) *proxyCtx {
	list, has := dict[subdomain]
	if has {
		for id, ctx := range list {
			p.del(dict, subdomain, id)
			return ctx
		}
	}
	return nil
}

func (p *proxy) add(dict map[string]map[string]*proxyCtx, subdomain, id string, ctx *proxyCtx) {
	if _, has := dict[subdomain]; !has {
		dict[ctx.subdomain] = map[string]*proxyCtx{}
	}

	dict[ctx.subdomain][ctx.id] = ctx
}

func (p *proxy) del(dict map[string]map[string]*proxyCtx, subdomain, id string) {
	delete(dict[subdomain], id)

	if len(dict[subdomain]) == 0 {
		delete(dict, subdomain)
	}
}

func (p *proxy) handler(ctx kit.GinContext) {
	if ctx.Request.Host == p.host {
		ctx.Status(200)

		subdomain := strings.Trim(ctx.Request.URL.Path, "/")
		if ctx.Request.Method == http.MethodGet {
			p.handleReq(subdomain, ctx)
			return
		}

		p.handleRes(subdomain, ctx)
		return
	}

	p.handleConsumer(ctx)
}

func (p *proxy) handleReq(subdomain string, ctx kit.GinContext) {
	wait, cancel := context.WithCancel(ctx.Request.Context())

	c := &proxyCtx{
		id:        kit.RandString(16),
		subdomain: subdomain,
		cancel:    cancel,
		ctx:       ctx,
	}

	p.req <- c

	<-wait.Done()

	p.reqLeave <- c
}

func (p *proxy) handleRes(subdomain string, ctx kit.GinContext) {
	id := ctx.GetHeader("Digto-ID")
	if id == "" {
		apiError(ctx, "Digto-ID header is not set")
		return
	}

	wait, cancel := context.WithCancel(ctx.Request.Context())

	c := &proxyCtx{
		id:        id,
		subdomain: subdomain,
		cancel:    cancel,
		ctx:       ctx,
	}
	p.res <- c

	<-wait.Done()

	p.resLeave <- c
}

func (p *proxy) handleConsumer(ctx kit.GinContext) {
	wait, cancel := context.WithCancel(ctx.Request.Context())
	subdomain := strings.Replace(ctx.Request.Host, "."+p.host, "", 1)
	id := kit.RandString(16)

	msg := &proxyCtx{
		subdomain: subdomain,
		id:        id,
		cancel:    cancel,
	}

	p.consumer <- msg

	<-wait.Done()

	msg.ctx.Header("Digto-ID", id)
	msg.ctx.Header("Digto-Method", ctx.Request.Method)
	msg.ctx.Header("Digto-URL", ctx.Request.URL.String())

	for k, l := range ctx.Request.Header {
		for _, v := range l {
			msg.ctx.Writer.Header().Add(k, v)
		}
	}
	msg.ctx.Writer.Header().Add("Host", ctx.Request.Host)

	_, err := io.Copy(msg.ctx.Writer, ctx.Request.Body)
	if err != nil {
		apiError(ctx, err.Error())
		apiError(msg.ctx, err.Error())
	}
	msg.cancel()

	wait, cancel = context.WithCancel(ctx.Request.Context())
	msg.cancel = cancel
	p.reqDone <- msg
	<-wait.Done()

	status := msg.ctx.GetHeader("Digto-Status")
	if status == "" {
		status = "200"
	}
	code, _ := strconv.ParseInt(status, 10, 32)
	ctx.Status(int(code))

	for k, l := range msg.ctx.Request.Header {
		if strings.HasPrefix(k, "Digto") {
			continue
		}
		for _, v := range l {
			ctx.Writer.Header().Add(k, v)
		}
	}

	_, err = io.Copy(ctx.Writer, msg.ctx.Request.Body)
	if err != nil {
		apiError(ctx, err.Error())
		apiError(msg.ctx, err.Error())
	}

	msg.cancel()

	p.consumerLeave <- msg
}

func (p *proxy) updateStatus() {
	p.status = map[string]interface{}{
		"reqConsumers": len(p.reqConsumers),
		"resConsumers": len(p.resConsumers),
		"reqWaitlist":  len(p.reqWaitlist),
		"resWaitlist":  len(p.resWaitlist),
	}
}
