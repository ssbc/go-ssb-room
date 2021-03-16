const pull = require('pull-stream')
const { readFileSync } = require('fs')


let newConnections = 0

module.exports = {
    secretStackPlugins: ['ssb-conn', 'ssb-room/tunnel/client'],

    before: (t, client, ready) => {
        // nothing to prepare (like publishes messages, or...)
        ready()
    },

    after: (t, client, roomSrvRpc, exit) => {
        newConnections++
        t.comment('new connection!' + roomSrvRpc.id)
        t.comment('total connections:' + newConnections)
        
        if (newConnections > 1) {
            t.comment('after call 2 - not exiting')
            return
        }
        // now connected to the room server
        
        // log all new endpoints
        pull(
            roomSrvRpc.tunnel.endpoints(),
            pull.drain(el => {
                t.comment("from roomsrv:",el)
            })
        )

        roomSrvRpc.tunnel.isRoom().then((yes) => {
            t.equal(yes, true, "expected isRoom to return true!")
            
            t.comment("peer is indeed a room!")

            // announce ourselves to the room/tunnel
            roomSrvRpc.tunnel.announce().then((ret) => {
                t.comment('announced!')

                // put there by the go test process
                let roomHandle = readFileSync('endpoint_through_room.txt').toString()
                t.comment("connecting to room handle:", roomHandle)

                client.conn.connect(roomHandle, (err, tunneldRpc) => {
                    t.error(err, "connected")
                    t.comment("got tunnel to:", tunneldRpc.id)

                    // check the tunnel connection works
                    tunneldRpc.tunnel.ping((err, timestamp) => {
                        t.error(err, "ping over the tunnel")
                        t.comment("ping:"+timestamp)

                        // start leaving after 1s
                        setTimeout(() => {
                            roomSrvRpc.tunnel.leave().then((ret) => {
                                t.comment('left room... exiting in 3s')
                                setTimeout(exit, 3000)
                            }).catch((err) => {
                                t.error(err, 'leave')
                            })
                        }, 1000)
                    })
                })
             
            }).catch((err) => {
                t.error(err, 'announce')
            })
        }).catch((err) => {
            t.error(err, 'isRoom failed')            
        })
    }
}