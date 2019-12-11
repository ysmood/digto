package server

import (
	"net/http"
	"time"

	"github.com/ysmood/ddns/adapters"
	"github.com/ysmood/digto/server/cert"
	"github.com/ysmood/kit"
	"github.com/ysmood/myip"
	"github.com/ysmood/storer"
)

type cache struct {
	store *storer.Value
}

var _ cert.Cache = &cache{}

func (c *cache) Get() ([]byte, error) {
	var data []byte
	err := c.store.Get(&data)
	return data, err
}

func (c *cache) Set(data []byte) error {
	return c.store.Set(&data)
}

func setupDNS(dnsProvider, dnsConfig, host string) error {
	if dnsConfig == "" {
		return nil
	}

	ip, err := myip.GetPublicIP()
	if err != nil {
		return err
	}
	dnsClient := adapters.New(dnsProvider, dnsConfig)
	err = dnsClient.SetRecord("*", host, ip)
	if err != nil {
		return err
	}
	err = dnsClient.SetRecord("@", host, ip)
	if err != nil {
		return err
	}
	return nil
}

func setupCert(host, dnsProvider, dnsConfig, caDirURL string, certCache *storer.Value) (*cert.Context, error) {
	if dnsConfig == "" {
		return nil, nil
	}

	cert, err := cert.New(
		host,
		dnsProvider,
		dnsConfig,
		caDirURL,
		&cache{certCache},
	)
	if err != nil {
		return nil, err
	}

	go func() {
		for {
			time.Sleep(24 * time.Hour)
			err := cert.Update()
			if err != nil {
				kit.Err("[digto]", err)
			}
		}
	}()

	return cert, nil

}

func apiError(ginCtx kit.GinContext, msg string) {
	ginCtx.Writer.Header().Set("Digto-Error", msg)
	ginCtx.AbortWithStatus(http.StatusBadRequest)
}
