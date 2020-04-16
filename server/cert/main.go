package cert

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"time"

	"crypto/elliptic"
	"crypto/rand"

	"github.com/go-acme/lego/v3/certcrypto"
	"github.com/go-acme/lego/v3/certificate"
	"github.com/go-acme/lego/v3/challenge"
	"github.com/go-acme/lego/v3/lego"
	"github.com/go-acme/lego/v3/providers/dns/dnspod"
	"github.com/go-acme/lego/v3/registration"
)

const month = time.Hour * 24 * 30

// Context ...
type Context struct {
	host         string
	key          *ecdsa.PrivateKey
	lastObtain   time.Time
	legoCert     *certificate.Resource
	cert         *tls.Certificate
	providerName string
	token        string
	cache        Cache

	caDirURL string
}

// Cache ...
type Cache interface {
	Get() ([]byte, error)
	Set([]byte) error
}

// New cache arg is optional
func New(host, providerName, token, caDirURL string, cache Cache) (*Context, error) {
	ctx := &Context{
		providerName: providerName,
		token:        token,
		host:         host,
		cache:        cache,
		caDirURL:     caDirURL,
	}

	var data []byte
	if cache != nil {
		var err error
		data, err = cache.Get()
		if err != nil {
			return nil, err
		}
	}

	if len(data) != 0 {
		err := ctx.unmarshal(data)
		if err != nil {
			return nil, err
		}
		if ctx.host != host || ctx.caDirURL != caDirURL {
			err = cache.Set(nil)
			if err != nil {
				return nil, err
			}
			return New(host, providerName, token, caDirURL, cache)
		}
	} else {
		key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			return nil, err
		}
		ctx.key = key
	}

	return ctx, ctx.Update()
}

// Update ...
func (ctx *Context) Update() error {
	if ctx.cert == nil {
		return ctx.obtain()
	} else if time.Since(ctx.lastObtain) > month {
		return ctx.renew()
	}
	return nil
}

// Cert ...
func (ctx *Context) Cert() *tls.Certificate {
	return ctx.cert
}

func (ctx *Context) obtain() error {
	request := certificate.ObtainRequest{
		Domains: []string{"*." + ctx.host, ctx.host},
		Bundle:  true,
	}
	client, err := ctx.client()
	if err != nil {
		return err
	}

	certificates, err := client.Certificate.Obtain(request)
	if err != nil {
		return err
	}
	ctx.legoCert = certificates
	ctx.lastObtain = time.Now()

	err = ctx.updateCert()
	if err != nil {
		return err
	}
	if ctx.cache != nil {
		return ctx.cache.Set(ctx.marshal())
	}
	return nil
}

func (ctx *Context) renew() error {
	client, err := ctx.client()
	if err != nil {
		return err
	}

	certificates, err := client.Certificate.Renew(*ctx.legoCert, true, false)
	if err != nil {
		return err
	}
	ctx.legoCert = certificates
	ctx.lastObtain = time.Now()

	err = ctx.updateCert()
	if err != nil {
		return err
	}
	if ctx.cache != nil {
		return ctx.cache.Set(ctx.marshal())
	}
	return nil
}

func (ctx *Context) updateCert() error {
	cert, err := tls.X509KeyPair(ctx.legoCert.Certificate, ctx.legoCert.PrivateKey)
	if err != nil {
		return err
	}
	ctx.cert = &cert
	return nil
}

func (ctx *Context) client() (*lego.Client, error) {
	config := lego.NewConfig(&user{key: ctx.key})

	if ctx.caDirURL != "" {
		config.CADirURL = ctx.caDirURL
	}
	config.Certificate.KeyType = certcrypto.RSA2048

	client, err := lego.NewClient(config)
	if err != nil {
		return nil, err
	}

	provider, err := ctx.getProvider()
	if err != nil {
		return nil, err
	}

	err = client.Challenge.SetDNS01Provider(provider)
	if err != nil {
		return nil, err
	}

	_, err = client.Registration.Register(registration.RegisterOptions{TermsOfServiceAgreed: true})
	if err != nil {
		return nil, err
	}

	return client, nil
}

func (ctx *Context) getProvider() (challenge.Provider, error) {
	switch ctx.providerName {
	case "dnspod":
		conf := dnspod.NewDefaultConfig()
		conf.LoginToken = ctx.token
		conf.HTTPClient.Timeout = 30 * time.Second
		return dnspod.NewDNSProviderConfig(conf)
	default:
		panic("provider not supported: " + ctx.providerName)
	}
}

type cacheData struct {
	Host              string
	CaDirURL          string
	LastObtain        time.Time
	Key               []byte
	Cert              certificate.Resource
	PrivateKey        []byte
	Certificate       []byte
	IssuerCertificate []byte
	CSR               []byte
}

func (ctx *Context) marshal() []byte {
	key, _ := x509.MarshalECPrivateKey(ctx.key)
	data, _ := json.Marshal(cacheData{
		Host:              ctx.host,
		CaDirURL:          ctx.caDirURL,
		LastObtain:        ctx.lastObtain,
		Key:               key,
		Cert:              *ctx.legoCert,
		PrivateKey:        ctx.legoCert.PrivateKey,
		Certificate:       ctx.legoCert.Certificate,
		IssuerCertificate: ctx.legoCert.IssuerCertificate,
		CSR:               ctx.legoCert.CSR,
	})
	return data
}

func (ctx *Context) unmarshal(data []byte) error {
	var cache cacheData
	err := json.Unmarshal(data, &cache)
	if err != nil {
		return err
	}

	key, err := x509.ParseECPrivateKey(cache.Key)
	if err != nil {
		return err
	}

	ctx.host = cache.Host
	ctx.caDirURL = cache.CaDirURL
	ctx.lastObtain = cache.LastObtain
	ctx.key = key
	ctx.legoCert = &cache.Cert
	ctx.legoCert.PrivateKey = cache.PrivateKey
	ctx.legoCert.Certificate = cache.Certificate
	ctx.legoCert.IssuerCertificate = cache.IssuerCertificate
	ctx.legoCert.CSR = cache.CSR

	err = ctx.updateCert()
	if err != nil {
		return err
	}

	return nil
}

type user struct {
	key crypto.PrivateKey
}

func (u *user) GetEmail() string {
	return "you@yours.com"
}
func (u user) GetRegistration() *registration.Resource {
	return nil
}
func (u *user) GetPrivateKey() crypto.PrivateKey {
	return u.key
}
