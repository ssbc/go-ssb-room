const pull = require('pull-stream')
const { readFileSync } = require('fs')


let newConnections = 0

module.exports = {
    secretStackPlugins: [
        'ssb-conn',
        'ssb-room-client',
    ],

    before: (t, client, ready) => {
        // nothing to prepare (like publishes messages, or...)
        ready()
    },

    after: (t, client, roomrpc, exit) => {
        newConnections++
        t.comment('client new connection!' + roomrpc.id)
        t.comment('total connections:' + newConnections)

        if (newConnections > 1) {
            t.comment('after call 2 - not doing anything')
            return
        }
        // now connected to the room server

        // log all new endpoints
        pull(
            roomrpc.tunnel.endpoints(),
            pull.drain(el => {
                t.comment("from roomsrv:" + JSON.stringify(el))
            })
        )

        // announce ourselves to the room/tunnel
        roomrpc.tunnel.announce().then((ret) => {
            t.comment('announced!')

            // put there by the go test process
            let roomHandle = readFileSync('endpoint_through_room.txt').toString()
            t.comment("connecting to room handle:", roomHandle)

            client.conn.connect(roomHandle, (err, tunneledrpc) => {
                t.error(err, 'connect through room')
                t.comment("got tunnel to:", tunneledrpc.id)

                // check the tunnel connection works
                tunneledrpc.testing.working((err, ok) => {
                    t.error(err, 'testing.working didnt error')
                    t.true(ok, 'testing.working is true')

                    // start leaving after 2s
                    setTimeout(() => {
                        roomrpc.tunnel.leave().then((ret) => {
                            t.comment('left room... exiting in 3s')
                            setTimeout(exit, 3000)
                        }).catch((err) => {
                            t.error(err, 'left leaving')
                        })
                    }, 2000)
                })
            })
        }).catch((err) => {
            t.error(err, 'announce on server')
        })
    }
}
