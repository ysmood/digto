const kit = require('nokit')

module.exports = (task) => {
  task('default', 'lint & test', async () => {
    await kit.spawn('eslint', ['.'])
    await kit.spawn('junit', ['-t', 30 * 1000, 'test.js'])
  })
}
