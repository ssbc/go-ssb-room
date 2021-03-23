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

    // t is the tape instance for assertions
    // sbot is the local sbot api
    // ready is a function to signal that preperation is done
    before: (t, sbot, ready) => {
        console.warn('before connect...')
        setTimeout(ready, 1000)
    },

    // t and sbot are same as above
    // clientRpc is the muxrpc client to the other remote (i.e a rpc handle for the room the client is connected to)
    // exit() is a function that needs to be called to halt the process and exit (it also calls t.end())
    after: (t, sbot, clientRpc, exit) => {
        console.warn('after connect...')

        setTimeout(exit, 5000)
    }
}