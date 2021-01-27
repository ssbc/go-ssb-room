const pull = require('pull-stream')

module.exports = {
    before: (sbot, ready) => {
        console.warn('before connect...')
        setTimeout(ready, 1000)
    },

    after: (sbot, exit) => {
        console.warn('after connect...')

        setTimeout(exit, 5000)
    }
}