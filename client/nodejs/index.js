const kit = require('nokit')

const defaultOpts = {
  scheme: 'https',
  apiHost: 'digto.org',
  subdomain: ''
}

const defaultSendOpts = {
  status: 200,
  headers: {}
}

module.exports = (opts) => {
  opts = kit._.defaults(opts, defaultOpts)

  if (!opts.subdomain) {
    throw new Error('subdomain option is empty')
  }

  return {
    publicUrl: () => {
      return `${opts.scheme}://${opts.subdomain}.${opts.apiHost}`
    },
    next: async (reqOpts) => {
      const url = `${opts.scheme}://${opts.apiHost}/${opts.subdomain}`

      const reqJob = kit.request(kit._.defaults({ url }, reqOpts))

      let id
      reqJob.req.on('response', (res) => { id = res.headers['digto-id'] })

      const res = await reqJob

      const send = async (sendOpts) => {
        sendOpts = kit._.defaults(sendOpts, defaultSendOpts)

        sendOpts.headers['digto-id'] = id
        sendOpts.headers['digto-status'] = sendOpts.status

        return kit.request({
          url,
          method: 'POST',
          headers: sendOpts.headers,
          reqData: sendOpts.body,
          body: false
        })
      }

      return [res, send]
    }

  }
}
