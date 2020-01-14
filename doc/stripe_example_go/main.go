package main

import (
	"context"

	"github.com/chromedp/chromedp"
	digto "github.com/ysmood/digto/client"
	"github.com/ysmood/kit"
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

	kit.Log("get card token:", token)

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

	kit.Log("get 3ds auth url:", url)

	// handle the callback from stripe
	go dig.One(func(ctx kit.GinContext) {
		ctx.Writer.WriteString(`<html><body id="done-3ds">ok</body></html>`)
	})

	ctx, _ := chromedp.NewContext(context.Background())

	kit.E(chromedp.Run(ctx,
		chromedp.Navigate(url),
	))
}

func req(url string) *kit.ReqContext {
	return kit.Req(url).Header("Authorization", "Bearer sk_test_4eC39HqLyjWDarjtT1zdp7dc")
}
