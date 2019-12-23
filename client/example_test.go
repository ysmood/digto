package client_test

import (
	"bytes"
	"fmt"
	"github.com/ysmood/digto/client"
	"io/ioutil"
)

func Example() {
	c := client.New("my-subdomain")

	req, res, _ := c.Next()

	data, _ := ioutil.ReadAll(req.Body)
	fmt.Println(string(data)) // output "my-data"

	_ = res(200, nil, bytes.NewBufferString("it works"))

	// curl https://my-subdomain.digto.org -d my-data
	// output "it works"
}
