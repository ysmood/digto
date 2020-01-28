package main

import (
	digto "github.com/ysmood/digto/client"
	"github.com/ysmood/kit"
	"github.com/ysmood/rod"
)

func main() {
	dig := digto.New(kit.RandString(16))

	token := req("https://api.stripe.com/v1/tokens").Post().Form(
		"card", map[string]interface{}{
			"number":    "4000000000003220",
			"exp_month": "7",
			"exp_year":  "2025",
			"cvc":       "314",
		},
	).MustJSON().Get("id").String()

	url := req("https://api.stripe.com/v1/payment_intents").Post().Form(
		"amount", "2000",
		"currency", "usd",
		"payment_method_data", map[string]interface{}{
			"type": "card",
			"card": map[string]interface{}{
				"token": token,
			},
		},
		"confirm", "true",
		"return_url", dig.PublicURL(),
	).MustJSON().Get("next_action.redirect_to_url.url").String()

	browser := rod.Open(nil)
	defer browser.Close()
	browser.Page(url).
		Element("[name=__privateStripeFrame4]").Frame().
		Element("#challengeFrame").Frame().
		Element("#test-source-authorize-3ds").Click()

	_, res, err := dig.Next()
	kit.E(err)
	kit.E(res(200, nil, nil))
}

func req(url string) *kit.ReqContext {
	return kit.Req(url).Header("Authorization", "Bearer sk_test_4eC39HqLyjWDarjtT1zdp7dc")
}
