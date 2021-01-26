const Path = require('path')
const { readFileSync } = require('fs')
const { loadOrCreateSync } = require('ssb-keys')
const pull = require('pull-stream') // used in eval scripts
const tape = require('tape')
const parallel = require('run-parallel') // used in eval scripts
const theStack = require('secret-stack')
const ssbCaps = require('ssb-caps')

const testSHSappKey = bufFromEnv('TEST_APPKEY')

let testAppkey = Buffer.from(ssbCaps.shs, 'base64')
if (testSHSappKey !== false) {
  testAppkey = testSHSappKey
}


const createSbot = theStack({caps: {shs: testAppkey } })
  .use(require('ssb-db'))
  .use(require('ssb-gossip'))
  .use(require('ssb-replicate'))
  .use(require('ssb-conn'))
  .use(require('ssb-room/tunnel/client'))

const testName = process.env.TEST_NAME
const testBob = process.env.TEST_BOB
const testAddr = process.env.TEST_GOADDR

const scriptBefore = readFileSync(process.env.TEST_BEFORE).toString()
const scriptAfter = readFileSync(process.env.TEST_AFTER).toString()

function bufFromEnv(evname) {
  const has = process.env[evname]
  if (has) {
    return Buffer.from(has, 'base64')
  }
  return false
}

tape.createStream().pipe(process.stderr)
tape(testName, function (t) {
  
  let timeoutLength = 15000
  var tapeTimeout = null
  function run() { // needs to be called by the before block when it's done
    t.timeoutAfter(timeoutLength) // doesn't exit the process
    tapeTimeout = setTimeout(() => {
      t.comment('test timeout')
      process.exit(1)
    }, timeoutLength*1.25)
    const to = `net:${testAddr}~shs:${testBob.substr(1).replace('.ed25519', '')}`
    t.comment('dialing:' + to)
    sbot.connect(to, (err) => {
      t.error(err, 'connected')
      eval(scriptAfter)
    })
  }

  function exit() { // call this when you're done
    sbot.close()
    t.comment('closed sbot')
    clearTimeout(tapeTimeout)
    t.end()
    process.exit(0)
  }

  const tempRepo = Path.join('testrun', testName)
  const keys = loadOrCreateSync(Path.join(tempRepo, 'secret'))
  const opts = {
    allowPrivate: true,
    path: tempRepo,
    keys: keys
  }

  if (testSHSappKey !== false) {
    opts.caps = opts.caps ? opts.caps : {}
    opts.caps.shs = testSHSappKey
  }

  const sbot = createSbot(opts)
  const alice = sbot.whoami()

  t.comment('sbot spawned, running before')
  console.log(alice.id) // tell go process who's incoming
  eval(scriptBefore)
})
