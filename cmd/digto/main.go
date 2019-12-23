package main

import (
	"github.com/ysmood/digto/server"
	"github.com/ysmood/kit"
)

func main() {
	app := kit.TasksNew("digto", "A service to help to expose http/https service to public network for integration test.")
	app.Version("v1.2.0")
	kit.Tasks().App(app).Add(
		kit.Task("serve", "start server").Init(serve),
	).Do()
}

func serve(cmd kit.TaskCmd) func() {
	cmd.Default()

	dbPath := cmd.Flag("db-path", "database path").Default("digto.db").String()
	dnsProvider := cmd.Flag("dns-provider", "dns provider name").Default("dnspod").String()
	dnsConfig := cmd.Flag("dns-config", "dns provider config").Short('c').Required().String()
	host := cmd.Flag("host", "host name").Short('h').Required().String()
	caDirURL := cmd.Flag("ca-dir-url", "acme ca dir url").Short('a').String()
	httpAddr := cmd.Flag("http-addr", "http address to listen to").Short('p').Default(":80").String()
	httpsAddr := cmd.Flag("https-addr", "https address to listen to").Short('s').Default(":443").String()

	return func() {
		s, err := server.New(*dbPath, *dnsProvider, *dnsConfig, *host, *caDirURL, *httpAddr, *httpsAddr)
		kit.E(err)
		kit.E(s.Serve())
	}
}
