// SPDX-FileCopyrightText: 2021 The NGI Pointer Secure-Scuttlebutt Team of 2020/2021
//
// SPDX-License-Identifier: MIT

const Path = require('path')
const tapSpec = require('tap-spec')
const tape = require('tape')
const { loadOrCreateSync } = require('ssb-keys')
const theStack = require('secret-stack')
const ssbCaps = require('ssb-caps')

const testSHSappKey = bufFromEnv('TEST_APPKEY')
let testAppkey = Buffer.from(ssbCaps.shs, 'base64')
if (testSHSappKey !== false) {
  testAppkey = testSHSappKey
}

stackOpts = {caps: {shs: testAppkey } }
let createSbot = theStack(stackOpts)
  .use(require('ssb-db2'))
  .use(require('ssb-db2/compat/db'))

const testName = process.env['TEST_NAME']
const testPort = process.env['TEST_PORT']
const testSession = require(process.env['TEST_SESSIONSCRIPT'])

const path = require("path")
const scriptname = path.basename(__filename)

// load the plugins needed for this session
for (plug of testSession.secretStackPlugins) {
  createSbot = createSbot.use(require(plug))
}

tape.createStream().pipe(tapSpec()).pipe(process.stderr);
tape(testName, function (t) {
  function comment (msg) {
    t.comment(`[${scriptname}] ${msg}`)
  }
// t.timeoutAfter(30000) // doesn't exit the process
//   const tapeTimeout = setTimeout(() => {
//     t.comment("test timeout")
//     process.exit(1)
//   }, 50000)

  function exit() { // call this when you're done
    sbot.close()
    comment(`closed server: ${testName}`)
    // clearTimeout(tapeTimeout)
    t.end()
    process.exit(0)
  }

  const tempRepo = process.env['TEST_REPO']
  console.warn("my repo:", tempRepo)
  const keys = loadOrCreateSync(Path.join(tempRepo, 'secret'))
  const sbot = createSbot({
    port: testPort,
    path: tempRepo,
    keys: keys,
  })
  const alice = sbot.whoami()

  comment("sbot spawned, running before")
 
  function ready() {
    comment(`server spawned, I am: ${alice.id}`)
    console.log(alice.id) // tell go process who our pubkey
  }
  testSession.before(t, sbot, ready)

  sbot.on("rpc:connect", (remote, isClient) => {
    comment(`new connection: ${remote.id}`)
    testSession.after(t, sbot, remote, exit)
  })
})

// util
function bufFromEnv(evname) {
  const has = process.env[evname]
  if (has) {
    return Buffer.from(has, 'base64')
  }
  return false
}
