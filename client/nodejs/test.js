const crypto = require('crypto')
const kit = require('nokit')
const digto = require('.')

module.exports = async it => {
  it('basic', async () => {
    const subdomain = crypto.randomBytes(8).toString('hex')

    const c = digto({ subdomain })

    const req = kit.request({ url: c.publicUrl(), reqData: 'ok' })
      .then((body) => {
        return it.eq(body, 'done')
      })

    const [res, send] = await c.next()

    await it.eq(res, 'ok')

    await send({ body: 'done' })

    await req
  })
}
