const pull = require('pull-stream')

module.exports = {
    secretStackPlugins: ['ssb-conn', 'ssb-room/tunnel/client'],

    before: (sbot, ready) => {
        ready()
    },

    after: (sbot, rpc, exit) => {
        sbot.on("rpc:connect", (remote, isClient) => {
            console.warn("tunneld connection to simple client!")

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
            }, 3000)
        })

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