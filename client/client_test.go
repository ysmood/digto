package client_test

import (
	"bytes"
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

	var senderRes string
	go func() {
		senderRes = kit.Req("http://" + host + "/path").Host(subdomain + ".digto.org").MustString()
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
	assert.Equal(t, "done", senderRes)
}

func TestServe(t *testing.T) {
	s, err := server.New("tmp/"+kit.RandString(16)+"/digto.db", "", "", "digto.org", "", ":0", "")
	kit.E(err)

	go func() { kit.E(s.Serve()) }()

	host := s.GetServer().Listener.Addr().String()

	subdomain := kit.RandString(16)
	wg := &sync.WaitGroup{}
	wg.Add(2)

	var senderRes string
	go func() {
		senderRes = kit.Req("http://" + host + "/path").Host(subdomain + ".digto.org").MustString()
		wg.Done()
	}()

	c := client.New(subdomain)
	c.APIHost = host
	c.APIScheme = "http"
	c.APIHeaderHost = "digto.org"

	path := ""
	go c.Serve(func(ctx kit.GinContext) {
		path = ctx.Request.URL.Path
		ctx.String(230, "done")
		wg.Done()
	})

	wg.Wait()

	assert.Equal(t, "/path", path)
	assert.Equal(t, "done", senderRes)
}
