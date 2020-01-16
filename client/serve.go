package client

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ysmood/kit"
)

// SchemeExec make the client accept and execute commands
const SchemeExec = "exec"

// One serves only one request with gin handler
func (c *Client) One(handler func(kit.GinContext)) error {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.NoRoute(handler)

	req, send, err := c.Next()

	if err != nil {
		return err
	}

	body := bytes.NewBuffer(nil)

	res := &response{
		status: http.StatusOK,
		header: http.Header{},
		body:   body,
	}

	engine.ServeHTTP(res, req)

	res.header.Add("Content-Length", fmt.Sprint(body.Len()))

	return send(res.status, res.header, body)
}

// Serve will proxy requests to the tcp address. Default scheme is http.
func (c *Client) Serve(addr, overrideHost, scheme string) {
	if scheme == "" {
		scheme = "http"
	}

	for {
		req, send, err := c.Next()
		if err != nil {
			log.Println(err)
			continue
		}

		go func() {
			log.Println("[access log]", kit.C(req.Method, "green"), req.URL.String())

			req.URL.Scheme = scheme
			req.URL.Host = addr
			if overrideHost != "" {
				req.Host = overrideHost
			}

			httpClient := &http.Client{}
			res, err := httpClient.Do(req)
			if err != nil {
				log.Println(err)
				return
			}

			err = send(res.StatusCode, res.Header, res.Body)
			if err != nil {
				log.Println(err)
			}
		}()
	}
}

// ServeExec run commands sent from
func (c *Client) ServeExec() {
	for {
		req, send, err := c.Next()
		if err != nil {
			log.Println(err)
			continue
		}

		go func() {
			raw, err := ioutil.ReadAll(req.Body)
			if err != nil {
				log.Println(err)
				return
			}

			data := kit.JSON(raw)
			args := []string{}
			if data.IsArray() {
				for _, arg := range data.Array() {
					args = append(args, arg.String())
				}
			} else {
				args = strings.Split(string(raw), " ")
			}

			if len(args) == 0 {
				return
			}

			cmd := exec.Command(args[0], args[1:]...)

			stdout, err := cmd.StdoutPipe()
			if err != nil {
				log.Println(err)
				return
			}
			stderr, err := cmd.StderrPipe()
			if err != nil {
				log.Println(err)
				return
			}
			out := io.MultiReader(stdout, stderr)

			err = cmd.Start()
			if err != nil {
				log.Println(err)
				return
			}

			err = send(http.StatusOK, nil, out)
			if err != nil {
				log.Println(err)
			}
		}()
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
