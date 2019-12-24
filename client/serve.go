package client

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ysmood/kit"
)

// One serve only one request
func (c *Client) One(handler func(kit.GinContext)) {
	engine := gin.New()
	engine.NoRoute(handler)

	req, send, err := c.Next()

	if err != nil {
		panic(err)
	}

	body := bytes.NewBuffer(nil)

	res := &response{
		status: http.StatusOK,
		header: http.Header{},
		body:   body,
	}

	engine.ServeHTTP(res, req)

	res.header.Add("Content-Length", fmt.Sprint(body.Len()))

	err = send(res.status, res.header, body)
	if err != nil {
		panic(err)
	}
}

type response struct {
	status int
	header http.Header
	body   io.Writer
}

func (res *response) Header() http.Header {
	return res.header
}

func (res *response) Write(data []byte) (int, error) {
	return res.body.Write(data)
}

func (res *response) WriteHeader(statusCode int) {
	res.status = statusCode
}
