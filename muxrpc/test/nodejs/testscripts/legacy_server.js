const pull = require('pull-stream')
const path = require("path")
const scriptname = path.basename(__filename)

let connections = 0

module.exports = {
    secretStackPlugins: ['ssb-conn', 'ssb-room/tunnel/server'],

    before: (t, sbot, ready) => {
        pull(
            sbot.conn.hub().listen(),
            pull.drain((p) => {
               t.comment(`[legacy-server.js] peer change ${p.type}: ${p.key}`)
            })
        )
        setTimeout(ready, 1000)
    },

    after: (t, sbot, client, exit) => {
        function comment (msg) {
          t.comment(`[${scriptname}] ${msg}`)
        }
        // this runs twice (for each connection)
        connections++
        comment(`new connection: ${client.id}`)
        comment(`total connections: ${connections}`)
        
        if (connections == 2) {
            t.comment('2nd connection received. exiting in 10 seconds')
            setTimeout(exit, 10000)
        }
    }
}
