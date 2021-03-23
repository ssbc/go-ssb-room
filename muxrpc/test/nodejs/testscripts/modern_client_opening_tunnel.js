const secretStackPlugins = require('./secretstack-modern') // use modern tunnel
const before = require('./minimal-before-setup')
const performOpeningTunnelTest = require('./client-opening-tunnel')

module.exports = {
    secretStackPlugins,
    before,
    after: performOpeningTunnelTest
}
