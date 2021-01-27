const pull = require('pull-stream')
const { readFileSync } = require('fs')

module.exports = {
    before: (sbot, ready) => {
        sbot.on('rpc:connect', rpc => {
            // log all new endpoints
            pull(
                rpc.tunnel.endpoints(),
                pull.drain(el => {
                    console.warn("from roomsrv:",el)
                })
            )

            rpc.tunnel.isRoom().then((yes) => {
                if (!yes) throw new Error("expected isRoom to be true!")
                console.warn("peer is indeed a room!")

                // announce ourselves to the room/tunnel
                rpc.tunnel.announce().then((ret) => {
                    console.warn('announced!')
                    console.warn(ret)
                
                    setTimeout(() => {
                        // put there by the go test process
                        let roomHandle = readFileSync('endpoint_through_room.txt').toString()
                        console.warn("connecting to room handle:", roomHandle)
        
                        sbot.connect(roomHandle, (err, tunneldRpc) => {
                            if (err) throw err
                            console.warn("got tunnel to:", tunneldRpc.id)
                        })
                    }, 5000)
                }).catch((err) => {
                    console.warn('announce failed')
                    throw err
                })
                
              

            }).catch((err) => {
                console.warn('isRoom failed')
                throw err
            })

         

            // leave after 5 seconds
            setTimeout(() => {
                rpc.tunnel.leave().then((ret) => {
                    console.warn('left')
                    console.warn(ret)
                }).catch((err) => {
                    console.warn('left failed')
                    throw err
                })
            }, 9000)
        })
        ready()
    },

    after: (sbot, exit) => {
        // now connected to the room
        console.warn('after connect... exiting in 10s')
        setTimeout(exit, 10000)
    }
}