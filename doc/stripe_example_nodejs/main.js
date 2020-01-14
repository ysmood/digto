let crypto = require('crypto')
let kit = require('nokit')
let digto = require('digto')
let qs = require('qs')
let puppeteer = require('puppeteer')

;(async () => {
    let dig = digto({ subdomain: crypto.randomBytes(8).toString('hex') })

    let headers = () => ({
        Authorization: 'Bearer sk_test_4eC39HqLyjWDarjtT1zdp7dc'
    })

    let token = JSON.parse(await kit.request({
        url: 'https://api.stripe.com/v1/tokens',
        method: 'post',
        headers: headers(),
        reqData: qs.stringify({
            card: {
                number:    '4000000000003220',
                exp_month: '7',
                exp_year:  '2025',
                cvc:       '314',
            }
        })
    })).id
    
    kit.logs("get card token:", token)

    let url = JSON.parse(await kit.request({
        url: 'https://api.stripe.com/v1/payment_intents',
        method: 'post',
        headers: headers(),
        reqData: qs.stringify({
            "amount": "2000",
            "currency": "usd",
            "payment_method_data": {
                "type": "card",
                "card": {
                    "token": token,
                },
            },
            "confirm": "true",
            "return_url": dig.publicUrl(),
        }),
    })).next_action.redirect_to_url.url

    kit.logs("get 3ds auth url:", url)

    let wait = dig.next({ body: false }).then(([res, send]) => {
        kit.logs('callback url:', res.headers['digto-url'])
        return send()
    })
    
    let browser = await puppeteer.launch();
    let page = await browser.newPage();
    await page.goto(url);
    await kit.sleep(5 * 1000)
    let button = await findInFrames(page, '#test-source-authorize-3ds')
    button.click()

    await wait

    kit.logs('done')

    await browser.close();
})()

async function recursiveFindInFrames(inputFrame, selector) {
    const frames = inputFrame.childFrames();
    const results = await Promise.all(
      frames.map(async frame => {
        const el = await frame.$(selector);
        if (el) return el;
        if (frame.childFrames().length > 0) {
          return await recursiveFindInFrames(frame, selector);
        }
        return null;
      })
    );
    return results.find(Boolean);
  }
  
async function findInFrames(page, selector) {
    const result = await recursiveFindInFrames(page.mainFrame(), selector);
    if (!result) {
        throw new Error(
            `The selector \`${selector}\` could not be found in any child frames.`
        );
    }
    return result;
}