const secretStackPlugins = require('./secretstack-modern')
const before = require('./minimal-before-setup')
const performClientTest = require('./client')
module.exports = {
    secretStackPlugins,
    before,
    after: performClientTest
}
