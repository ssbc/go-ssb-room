/*
this is a tempalte for a script to be used in the go<>js tests.

all the setup of the peers is done in sbot_client and sbot_server js.

warning: only log to stderr (console.warn)
DONT log to stdout (console.log) as this is connected to the go test process for initialization

TODO: pass the tape instance into the module, so that t.error and it's other helpers can be used.
proably by turning the exported object into an init function which returns the {before, after} object.
*/

// const pull = require('pull-stream')

module.exports = {
    secretStackPlugins: ['ssb-blobs', 'ssb-what-ever-you-need'],

    before: (sbot, ready) => {
        console.warn('before connect...')
        setTimeout(ready, 1000)
    },

    after: (sbot, exit) => {
        console.warn('after connect...')

        setTimeout(exit, 5000)
    }
}