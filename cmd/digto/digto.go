package main

import (
	"io"
	"log"
	"os"

	"github.com/ysmood/digto/client"
	"github.com/ysmood/digto/server"
	"github.com/ysmood/kit"
)

func main() {
	app := kit.TasksNew("digto", "A service to help to expose http/https service to public network for integration test.")
	app.Version(server.Version)
	kit.Tasks().App(app).Add(
		kit.Task("serve", "start server").Init(serve),
		kit.Task("proxy", "proxy a subdomain to the tcp address").Init(proxy),
		kit.Task("exec", "run a command on remote").Init(exec),
	).Do()
}

func serve(cmd kit.TaskCmd) func() {
	dbPath := cmd.Flag("db-path", "database path").Default("digto.db").String()
	dnsProvider := cmd.Flag("dns-provider", "dns provider name").Default("dnspod").String()
	dnsConfig := cmd.Flag("dns-config", "dns provider config").Short('c').Required().String()
	host := cmd.Flag("host", "host name").Short('h').Required().String()
	caDirURL := cmd.Flag("ca-dir-url", "acme ca dir url").Short('a').String()
	httpAddr := cmd.Flag("http-addr", "http address to listen to").Short('p').Default(":80").TCP()
	httpsAddr := cmd.Flag("https-addr", "https address to listen to").Short('s').Default(":443").TCP()
	timeout := cmd.Flag("timeout", "global http timeout").Short('o').Default("2m").Duration()

	return func() {
		s, err := server.New(*dbPath, *dnsProvider, *dnsConfig, *host, *caDirURL, (*httpAddr).String(), (*httpsAddr).String(), *timeout)
		kit.E(err)
		kit.E(s.Serve())
	}
}

func proxy(cmd kit.TaskCmd) func() {
	cmd.Default()

	subdomain := cmd.Arg("subdomain", "the subdomain to use, default is random string").String()
	addr := cmd.Arg("addr", "the tcp address to proxy to").Default(":3000").TCP()
	hostHeader := cmd.Arg("host-header", "override the host header when making request to addr").String()
	scheme := cmd.Flag("scheme", "scheme to use when send request to addr").Short('s').Default("http").Enum(
		"http", "https", client.SchemeExec,
	)

	return func() {
		if *subdomain == "" {
			*subdomain = kit.RandString(4)
		}

		c := client.New(*subdomain)

		addr := (*addr).String()

		if *scheme == client.SchemeExec {
			kit.Log("digto client:", *subdomain)
			c.ServeExec()
		} else {
			kit.Log("digto client:", c.PublicURL(), kit.C("->", "cyan"), addr)
			c.Serve(addr, *hostHeader, *scheme)
		}
	}
}

func exec(cmd kit.TaskCmd) func() {
	subdomain := cmd.Arg("subdomain", "the subdomain to use, default is random string").Required().String()
	args := cmd.Arg("cmd", "the tcp address to proxy to").Required().Strings()

	return func() {
		c := client.New(*subdomain)
		res, err := c.Exec(*args...)
		if err != nil {
			log.Println(err)
			return
		}
		_, err = io.Copy(os.Stdout, res)
		if err != nil {
			log.Println(err)
		}
	}
}
