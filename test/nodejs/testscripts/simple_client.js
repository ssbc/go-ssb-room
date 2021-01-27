const pull = require('pull-stream')

module.exports = {
    before: (sbot, ready) => {
        sbot.on('rpc:connect', rpc => {
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

            // leave after 5 seconds
            setTimeout(() => {
                rpc.tunnel.leave().then((ret) => {
                    console.warn('left')
                    console.warn(ret)
                }).catch((err) => {
                    console.warn('left failed')
                    throw err
                })
            }, 4000)
        })
        ready()
    },

    after: (sbot, exit) => {
        console.warn('after connect... exiting in 10s')
        setTimeout(exit, 10000)
    }
}