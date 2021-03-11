const pull = require('pull-stream')

module.exports = {
    before: (sbot, ready) => {
        pull(
            sbot.conn.hub().listen(),
            pull.drain((p) => {
                console.warn('peer change:',p.type, p.key)
            })
        )
        setTimeout(ready, 1000)
    },

    after: (sbot, client, exit) => {
        // hrm.. this runs twice (for each connection)
        console.warn('server new connection:', client.id)
        setTimeout(exit, 30000)
    }
}