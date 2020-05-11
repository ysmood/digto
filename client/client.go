package client

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/ysmood/kit"
)

// Client ...
type Client struct {
	// Scheme default is https
	Scheme string

	// Subdomain ...
	Subdomain string

	// APIScheme to use for api request
	APIScheme string
	// APIHost api host
	APIHost string
	// Host api header host
	APIHeaderHost string

	httpClient *http.Client

	Log func(...interface{})
}

// Send response back to the public request
type Send func(status int, header http.Header, body io.Reader) error

// New creates a client with default config
func New(subdomain string) *Client {
	return &Client{
		Scheme:        "https",
		APIScheme:     "https",
		APIHost:       "digto.org",
		APIHeaderHost: "digto.org",
		Subdomain:     subdomain,
		httpClient:    &http.Client{},
		Log: func(s ...interface{}) {
			log.Println(s...)
		},
	}
}

// PublicURL returns the url exposed to public
func (c *Client) PublicURL() string {
	return c.Scheme + "://" + c.Subdomain + "." + c.APIHost
}

// Next gets the next request from public
func (c *Client) Next() (*http.Request, Send, error) {
	apiURL := url.URL{
		Scheme: c.APIScheme,
		Host:   c.APIHost,
		Path:   c.Subdomain,
	}

	senderRes, err := resError(kit.Req(apiURL.String()).Client(c.httpClient).Host(c.APIHeaderHost).Response())
	if err != nil {
		return nil, nil, err
	}

	receiverReq, err := http.NewRequestWithContext(
		senderRes.Request.Context(),
		senderRes.Header.Get("Digto-Method"),
		c.PublicURL()+senderRes.Header.Get("Digto-URL"),
		senderRes.Body,
	)
	if err != nil {
		return nil, nil, err
	}

	for k, v := range senderRes.Header {
		if !strings.HasPrefix(k, "Digto") {
			receiverReq.Header[k] = v
		}
	}

	receiverReq.Host = senderRes.Header.Get("Host")

	send := func(status int, header http.Header, body io.Reader) error {
		headerToSend := []string{
			"Digto-ID", senderRes.Header.Get("Digto-ID"),
			"Digto-Status", fmt.Sprint(status),
		}
		if header != nil {
			for k, l := range header {
				for _, v := range l {
					headerToSend = append(headerToSend, k, v)
				}
			}
		}

		_, err = resError(
			kit.Req(apiURL.String()).Post().
				Client(c.httpClient).
				Host(c.APIHeaderHost).
				Header(headerToSend...).Body(body).Response(),
		)
		return err
	}

	return receiverReq, send, nil
}

func resError(res *http.Response, err error) (*http.Response, error) {
	if err != nil {
		return nil, err
	}

	errMsg := res.Header.Get("Digto-Error")
	if errMsg != "" {
		return res, errors.New(errMsg)
	}

	return res, err
}
