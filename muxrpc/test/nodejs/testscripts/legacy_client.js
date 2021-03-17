const pull = require('pull-stream')

module.exports = {
    secretStackPlugins: [
        'ssb-conn',
        'ssb-room/tunnel/client',
    ],

    before: (t, sbot, ready) => {
        ready()
    },

    after: (t, sbot, rpc, exit) => {
        // this waits for a new incomming connection _after_ the room server is connected already
        // so it will be an incomming tunnel client.
        // since this calls exit() - if no client connects it will not exit
        sbot.on("rpc:connect", (remote, isClient) => {
            console.warn("tunneld connection to simple client!")

            // leave after 3 seconds (give the other party time to call ping)
            setTimeout(() => {
                rpc.tunnel.leave().then((ret) => {
                    console.warn('left')
                    console.warn(ret)
                    console.warn('room left... exiting in 1s')
                    setTimeout(exit, 1000)
                }).catch((err) => {
                    t.error(err, 'tunnel.leave failed')
                })
            }, 3000)
        })

        // announce ourselves to the room/tunnel
        rpc.tunnel.announce().then((ret) => {
            console.warn('announced!')
            console.warn(ret)
        }).catch((err) => {
            t.error(err, 'tunnel.announce failed')
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