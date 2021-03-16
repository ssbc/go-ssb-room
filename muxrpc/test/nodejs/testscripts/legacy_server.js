const pull = require('pull-stream')

let connections = 0

module.exports = {
    secretStackPlugins: ['ssb-conn', 'ssb-room/tunnel/server'],

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
        // this runs twice (for each connection)
        connections++
        console.warn('server new connection:', client.id, connections)
        
        if (connections == 2) {
            setTimeout(exit, 15000)
        }
    }
}