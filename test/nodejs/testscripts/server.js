const pull = require('pull-stream')

module.exports = {
    before: (sbot, ready) => {
        pull(
            sbot.conn.peers(),
            pull.drain((p) => {
                console.warn('peer change:',p)
            })
        )

        setTimeout(ready, 1000)
        // ready()
    },

    after: (sbot, client, exit) => {
        console.warn('after:', sbot.id, client.id)

        setTimeout(exit, 5000)
    }
}