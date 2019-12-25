package client_test

import (
	"bytes"
	"net/http"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ysmood/digto/client"
	"github.com/ysmood/digto/server"
	"github.com/ysmood/kit"
)

func TestBasic(t *testing.T) {
	s, err := server.New("tmp/"+kit.RandString(16)+"/digto.db", "", "", "digto.org", "", ":0", "")
	kit.E(err)

	go func() { kit.E(s.Serve()) }()

	host := s.GetServer().Listener.Addr().String()

	subdomain := kit.RandString(16)
	wg := &sync.WaitGroup{}
	wg.Add(1)

	go func() {
		senderRes := kit.Req("http://" + host + "/path").Host(subdomain + ".digto.org").MustString()
		assert.Equal(t, "done", senderRes)

		wg.Done()
	}()

	c := client.New(subdomain)
	c.APIHost = host
	c.APIScheme = "http"
	c.APIHeaderHost = "digto.org"

	assert.Equal(t, "https://"+subdomain+"."+host, c.PublicURL())

	req, send, err := c.Next()
	kit.E(err)

	assert.Equal(t, "/path", req.URL.Path)

	kit.E(send(200, nil, bytes.NewBufferString("done")))

	wg.Wait()
}

func TestOne(t *testing.T) {
	s, err := server.New("tmp/"+kit.RandString(16)+"/digto.db", "", "", "digto.org", "", ":0", "")
	kit.E(err)

	go func() { kit.E(s.Serve()) }()

	host := s.GetServer().Listener.Addr().String()

	subdomain := kit.RandString(16)
	wg := &sync.WaitGroup{}
	wg.Add(2)

	go func() {
		senderRes := kit.Req("http://" + host + "/path").Host(subdomain + ".digto.org").MustString()
		assert.Equal(t, "done", senderRes)

		wg.Done()
	}()

	c := client.New(subdomain)
	c.APIHost = host
	c.APIScheme = "http"
	c.APIHeaderHost = "digto.org"

	kit.E(c.One(func(ctx kit.GinContext) {
		path := ctx.Request.URL.Path
		assert.Equal(t, "/path", path)
		ctx.String(230, "done")
		wg.Done()
	}))

	wg.Wait()
}

func TestServe(t *testing.T) {
	s, err := server.New("tmp/"+kit.RandString(16)+"/digto.db", "", "", "digto.org", "", ":0", "")
	kit.E(err)

	go func() { kit.E(s.Serve()) }()

	host := s.GetServer().Listener.Addr().String()

	subdomain := kit.RandString(16)
	wg := &sync.WaitGroup{}
	wg.Add(2)

	go func() {
		senderRes := kit.Req("http://" + host + "/path").Host(subdomain + ".digto.org").MustString()
		assert.Equal(t, "done "+subdomain+".digto.org", senderRes)

		wg.Done()
	}()

	c := client.New(subdomain)
	c.APIHost = host
	c.APIScheme = "http"
	c.APIHeaderHost = "digto.org"

	srv := kit.MustServer(":0")

	srv.Engine.GET("/path", func(ctx kit.GinContext) {
		ctx.String(http.StatusOK, "done "+ctx.Request.Host)
		wg.Done()
	})

	go srv.MustDo()

	go c.Serve(srv.Listener.Addr().String(), "")

	wg.Wait()
}
