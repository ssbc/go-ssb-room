// SPDX-FileCopyrightText: 2021 The NGI Pointer Secure-Scuttlebutt Team of 2020/2021
//
// SPDX-License-Identifier: MIT

const secretStackPlugins = require('./secretstack-modern')
const before = require('./minimal-before-setup')

module.exports = {
  secretStackPlugins,
  before,
  after: (t, sbot, rpc, exit) => {

    // give ssb-conn a moment to settle
    setTimeout(() => {

      sbot.roomClient.registerAlias(rpc.id, "alice", (err, ret) => {
        t.error(err, 'registerAlias')
        t.ok(ret)
        t.equals(typeof ret, 'string')
        t.ok(new URL(ret))
        
        sbot.roomClient.revokeAlias(rpc.id, "alice", (err, ret) => {
          t.error(err, 'revokeAlias')
          t.comment(`revokeAlias value: ${ret}`)
          exit()
        })
      })

    }, 1000)
  }
}
