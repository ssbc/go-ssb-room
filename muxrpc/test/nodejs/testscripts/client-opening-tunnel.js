// SPDX-FileCopyrightText: 2021 The NGI Pointer Secure-Scuttlebutt Team of 2020/2021
//
// SPDX-License-Identifier: MIT

const pull = require('pull-stream')
const { readFileSync } = require('fs')
const path = require("path")
const scriptname = path.basename(__filename)

let newConnections = 0

module.exports = (t, client, roomrpc, exit) => {
  // shadow t.comment to include file making the comment
  function comment (msg) {
    t.comment(`[${scriptname}] ${msg}`)
  }
  newConnections++
  comment(`new connection: ${roomrpc.id}`)
  comment(`total connections: ${newConnections}`)

  if (newConnections > 1) {
    comment('more than two connnections, not doing anything')
    return
  }

  // we are now connected to the room server.
  // log all new endpoints
  pull(
    roomrpc.tunnel.endpoints(),
    pull.drain(el => {
      comment(`from roomsrv: ${JSON.stringify(el)}`)
    }, (err) => {
      t.comment('endpoints closed', err)
    })
  )

  // give the room time to start
  setTimeout(() => {
    // announce ourselves to the room/tunnel
    roomrpc.tunnel.announce((err, ret) => {
      t.error(err, 'announce on server')
      comment('announced!')

      // put there by the go test process
      let roomHandle = readFileSync('endpoint_through_room.txt').toString()
      comment(`connecting to room handle: ${roomHandle}`)

      client.conn.connect(roomHandle, (err, tunneledrpc) => {
        t.error(err, 'connect through room')
        comment(`got a tunnel to: ${tunneledrpc.id}`)

        // check the tunnel connection works
        tunneledrpc.testing.working((err, ok) => {
          t.error(err, 'testing.working didnt error')
          t.true(ok, 'testing.working is true')

          // start leaving after 2s
          setTimeout(() => {
            roomrpc.tunnel.leave((err, ret) => {
              t.error(err, 'tunnel.leave')
              comment('left room... exiting in 1s')
              setTimeout(exit, 1000)
            })
          }, 2000)
        })
      })
    })
  }, 5000)
}
