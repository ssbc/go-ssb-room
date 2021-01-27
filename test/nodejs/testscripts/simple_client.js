const pull = require('pull-stream')

module.exports = {
    before: (sbot, ready) => {
        sbot.on('rpc:connect', rpc => {
            var ret = rpc.tunnel.announce()
            console.warn('announced')
            console.warn(ret)
            pull(
                rpc.tunnel.endpoints(),
                pull.drain(el => {
                    console.warn("from roomsrv:",el)
                })
            )

            setTimeout(() => {
                ret = rpc.tunnel.leave()
                console.warn('left')
                console.warn(ret)
            }, 2500)
        })
        ready()
    },

    after: (sbot, exit) => {
        console.warn('after connect...')

        setTimeout(exit, 5000)
    }
}