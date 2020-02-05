package client

import (
	"bytes"
	"errors"
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

		go c.serve(addr, overrideHost, scheme, req, send)
	}
}

func (c *Client) serve(addr, overrideHost, scheme string, req *http.Request, send Send) {
	log.Println("[access log]", kit.C(req.Method, "green"), req.URL.String())

	req.URL.Scheme = scheme
	req.URL.Host = addr
	if overrideHost != "" {
		req.Host = overrideHost
	}

	httpClient := &http.Client{}
	res, err := httpClient.Do(req)
	if err != nil {
		resErr(send, err.Error())
		return
	}

	err = send(res.StatusCode, res.Header, res.Body)
	if err != nil {
		log.Println(err)
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

		go c.serveExec(req, send)
	}
}

func (c *Client) serveExec(req *http.Request, send Send) {
	raw, err := ioutil.ReadAll(req.Body)
	if err != nil {
		resErr(send, err.Error())
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

	if len(args) == 0 || args[0] == "" {
		resErr(send, "empty args")
		return
	}

	if isBuiltin, fn := c.execBuiltin(args); isBuiltin {
		err := fn()
		if err != nil {
			resErr(send, err.Error())
		}
		err = send(http.StatusOK, nil, nil)
		if err != nil {
			resErr(send, err.Error())
		}
		return
	}

	cmd := exec.Command(args[0], args[1:]...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		resErr(send, err.Error())
		return
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		resErr(send, err.Error())
		return
	}
	out := io.MultiReader(stdout, stderr)

	err = cmd.Start()
	if err != nil {
		resErr(send, err.Error())
		return
	}

	err = send(http.StatusOK, nil, out)
	if err != nil {
		resErr(send, err.Error())
	}
}

func (c *Client) execBuiltin(args []string) (bool, func() error) {
	switch args[0] {
	case "writefile":
		return true, func() error {
			if len(args) < 3 {
				return errors.New("writefile requires 2 args")
			}
			return kit.OutputFile(args[1], args[2], nil)
		}
	default:
		return false, nil
	}
}

func resErr(send Send, msg string) {
	err := send(http.StatusInternalServerError, nil, bytes.NewBufferString(msg))
	if err != nil {
		log.Println(err)
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
