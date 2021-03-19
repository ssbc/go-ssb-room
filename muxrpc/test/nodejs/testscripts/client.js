const pull = require('pull-stream')

module.exports = (t, sbot, rpc, exit) => {
  // this waits for a new incoming connection _after_ the room server is connected already
  // so it will be an incomming tunnel client.
  // since this calls exit() - if no client connects it will not exit
  sbot.on("rpc:connect", (remote, isClient) => {
    t.comment("tunneled connection to simple client!")

    // leave after 3 seconds (give the other party time to call `testing.working()`)
    setTimeout(() => {
      rpc.tunnel.leave((err, ret) => {
        t.error(err, 'tunnel.leave')
        t.comment(`tunnel error: ${err}`)
        t.comment(`leave value: ${ret}`)
        t.comment('left')
        t.comment('room left... exiting in 1s')
        setTimeout(exit, 1000)
      })
    }, 3000)
  })

  // announce ourselves to the room/tunnel
  rpc.tunnel.announce((err, ret) => {
    t.error(err, 'tunnel.announce')
    t.comment(`announce error: ${err}`)
    t.comment(`announce value: ${ret}`)
    t.comment('announced!')
  })

  // log all new endpoints
  pull(
    rpc.tunnel.endpoints(),
    pull.drain(el => {
      t.comment("from roomsrv:",el)
    })
  )
}
