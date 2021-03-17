const pull = require('pull-stream')

module.exports = {
    secretStackPlugins: [
        'ssb-conn',
        'ssb-room-client',
    ],

    before: (t, sbot, ready) => {
        ready()
    },

    after: (t, sbot, rpc, exit) => {
        sbot.on("rpc:connect", (remote, isClient) => {
            console.warn("tunneld connection to simple client!")

            // check the tunnel connection works
            remote.testing.working((err, ok) => {
                t.error(err, 'testing.working didnt error')
                t.true(ok, 'testing.working is true')

                
                // leave after 5 seconds
                setTimeout(() => {
                    rpc.tunnel.leave().then((ret) => {
                        console.warn('left')
                        console.warn(ret)
                        console.warn('room left... exiting in 10s')
                        setTimeout(exit, 10000)
                    }).catch((err) => {
                        console.warn('left failed')
                        throw err
                    })
                }, 5000)
            })
        }) // on rpc:connect

        // announce ourselves to the room/tunnel
        rpc.tunnel.announce().then((ret) => {
            console.warn('announced!')
            console.warn(ret)
        }).catch((err) => {
            console.warn('announce failed')
            throw err
        })

        // log all new endpoints
        pull(
            rpc.tunnel.endpoints(),
            pull.drain(el => {
                console.warn("from roomsrv:",el)
            })
        )
    }
}