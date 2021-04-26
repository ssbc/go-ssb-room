const pull = require('pull-stream')
const path = require("path")
const scriptname = path.basename(__filename)

module.exports = (t, sbot, rpc, exit) => {
  // shadow t.comment to include file making the comment
  function comment (msg) {
    t.comment(`[${scriptname}] ${msg}`)
  }
  // this waits for a new incoming connection _after_ the room server is connected already
  // so it will be an incomming tunnel client.
  // since this calls exit() - if no client connects it will not exit
  sbot.on("rpc:connect", (remote, isClient) => {
    comment("tunneled connection to simple client!")

    // leave after 3 seconds (give the other party time to call `testing.working()`)
    setTimeout(() => {
      rpc.tunnel.leave((err, ret) => {
        t.error(err, 'tunnel.leave')
        comment(`tunnel error: ${err}`)
        comment(`leave value: ${ret}`)
        comment('left, exiting in 3s')
        setTimeout(exit, 3000)
      })
    }, 1000)
  })

  // announce ourselves to the room/tunnel
  rpc.tunnel.announce((err, ret) => {
    t.error(err, 'tunnel.announce')
    comment(`announce error: ${err}`)
    comment(`announce value: ${ret}`)
    comment('announced!')
  })

  // log all new endpoints
  pull(
    rpc.tunnel.endpoints(),
    pull.drain(el => {
      comment(`from roomsrv: ${el}`)
    }, (err) => {
      t.comment('endpoints closed', err)
    })
  )
}
