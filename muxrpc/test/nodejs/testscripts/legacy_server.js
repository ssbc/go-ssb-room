const pull = require('pull-stream')

let connections = 0

module.exports = {
    secretStackPlugins: ['ssb-conn', 'ssb-room/tunnel/server'],

    before: (t, sbot, ready) => {
        pull(
            sbot.conn.hub().listen(),
            pull.drain((p) => {
               t.comment(`peer change ${p.type}: ${p.key}`)
            })
        )
        setTimeout(ready, 1000)
    },

    after: (t, sbot, client, exit) => {
        // this runs twice (for each connection)
        connections++
        t.comment(`server new connection: ${client.id}`)
        t.comment(`total connections: ${connections}`)
        
        if (connections == 2) {
            t.comment('2nd connection received. exiting in 15 seconds')
            setTimeout(exit, 15000)
        }
    }
}