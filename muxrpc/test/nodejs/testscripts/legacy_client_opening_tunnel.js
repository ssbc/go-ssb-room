const pull = require('pull-stream')
const { readFileSync } = require('fs')


let newConnections = 0

module.exports = {
    secretStackPlugins: ['ssb-conn', 'ssb-room/tunnel/client'],

    before: (client, ready) => {
        // nothing to prepare (like publishes messages, or...)
        ready()

        // let msg = {
        //     type: 'test',
        // }
        // client.publish(msg, (err) => {
        //     if (err) throw err
        // })
    },

    after: (client, roomSrvRpc, exit) => {
        newConnections++
        console.warn('new connection!', roomSrvRpc.id, 'total:', newConnections)
        
        if (newConnections > 1) {
            console.warn('after call 2 - not exiting')
            return
        }
        // now connected to the room server
        
        // log all new endpoints
        pull(
            roomSrvRpc.tunnel.endpoints(),
            pull.drain(el => {
                console.warn("from roomsrv:",el)
            })
        )

        roomSrvRpc.tunnel.isRoom().then((yes) => {
            if (!yes) throw new Error("expected isRoom to be true!")
            console.warn("peer is indeed a room!")

            // announce ourselves to the room/tunnel
            roomSrvRpc.tunnel.announce().then((ret) => {
                console.warn('announced!')

                // put there by the go test process
                let roomHandle = readFileSync('endpoint_through_room.txt').toString()
                console.warn("connecting to room handle:", roomHandle)

                client.conn.connect(roomHandle, (err, tunneldRpc) => {
                    if (err) throw err
                    console.warn("got tunnel to:", tunneldRpc.id)

                    // check the tunnel connection works
                    tunneldRpc.tunnel.ping((err, id) => {
                        if (err) throw err
                        console.warn("ping:", id)

                        // start leaving after 2s
                        setTimeout(() => {
                            roomSrvRpc.tunnel.leave().then((ret) => {
                                console.warn('left room... exiting in 3s')
                                setTimeout(exit, 3000)
                            }).catch((err) => {
                                console.warn('left failed')
                                throw err
                            })
                        }, 2000)
                    })
                })
             
            }).catch((err) => {
                console.warn('announce failed')
                throw err
            })

        }).catch((err) => {
            console.warn('isRoom failed')
            throw err
        })
    }
}