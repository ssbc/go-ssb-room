const Path = require('path')
const { loadOrCreateSync } = require('ssb-keys')
const tape = require('tape')
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

// the other peer we are talking to
const testPeerAddr = process.env.TEST_PEERADDR
const testPeerRef = process.env.TEST_PEERREF

const testSession = require(process.env['TEST_SESSIONSCRIPT'])

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
  function ready() { // needs to be called by the before block when it's done
    t.timeoutAfter(timeoutLength) // doesn't exit the process
    tapeTimeout = setTimeout(() => {
      t.comment('test timeout')
      process.exit(1)
    }, timeoutLength*1.25)
    const to = `net:${testPeerAddr}~shs:${testPeerRef.substr(1).replace('.ed25519', '')}`
    t.comment('dialing:' + to)
    sbot.connect(to, (err) => {
      t.error(err, 'connected')
      testSession.after(sbot, exit)
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
  testSession.before(sbot, ready)
})
