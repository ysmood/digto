package server_test

import (
	"math/rand"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/ysmood/digto/server"
	"github.com/ysmood/kit"
)

func TestBasic(t *testing.T) {
	dir := "tmp/" + kit.RandString(16)

	s, err := server.New(dir+"/digto.db", "", "", "digto.org", "", ":0", "")
	kit.E(err)

	go func() { kit.E(s.Serve()) }()

	host := "http://" + s.GetServer().Listener.Addr().String()

	body := ""
	status := 0
	wait := make(chan kit.Nil)
	header := ""

	go func() {
		req := kit.Req(host+"/path").
			Host("a.digto.org").
			Header("My", "header").
			Post().StringBody("data")
		req.MustDo()

		body = req.MustString()
		status = req.MustResponse().StatusCode
		header = req.MustResponse().Header.Get("Header")

		wait <- kit.Nil{}
	}()

	req := kit.Req(host + "/a").Host("digto.org")

	assert.Equal(t, "data", req.MustString())
	assert.Equal(t, "POST", req.MustResponse().Header.Get("Digto-Method"))
	assert.Equal(t, "/path", req.MustResponse().Header.Get("Digto-URL"))
	assert.Equal(t, "header", req.MustResponse().Header.Get("My"))
	assert.Equal(t, 200, req.MustResponse().StatusCode)

	res := kit.Req(host+"/a").
		Post().StringBody("test").
		Host("digto.org").
		Header(
			"Digto-ID", req.MustResponse().Header.Get("Digto-ID"),
			"Digto-Status", "230",
			"Header", "value",
		).
		MustResponse()

	assert.Equal(t, 200, res.StatusCode)

	<-wait

	assert.Equal(t, "test", body)
	assert.Equal(t, 230, status)
	assert.Equal(t, "value", header)
}

func TestConcurent(t *testing.T) {
	dir := "tmp/" + kit.RandString(16)

	srv, err := server.New(dir+"/digto.db", "", "", "digto.org", "", ":0", "")
	kit.E(err)

	go func() { kit.E(srv.Serve()) }()

	host := "http://" + srv.GetServer().Listener.Addr().String()

	wg := &sync.WaitGroup{}

	send := func(subdomain string) {
		s := kit.Req(host).Host(subdomain + ".digto.org").MustString()
		if s != subdomain {
			panic("res doesn't match")
		}
		wg.Done()
	}

	read := func(subdomain string) {
		req := kit.Req(host + "/" + subdomain).Host("digto.org")

		kit.Req(host+"/"+subdomain).Post().Host("digto.org").
			StringBody(subdomain).
			Header(
				"Digto-ID", req.MustResponse().Header.Get("Digto-ID"),
			).MustDo()

		wg.Done()
	}

	const n = 10
	wg.Add(n * 2)

	subdomains := []string{"a", "b", "c"}
	for range make([]kit.Nil, n) {
		subdomain := subdomains[rand.Intn(2)]
		go send(subdomain)
		go read(subdomain)
	}

	wg.Wait()
}
