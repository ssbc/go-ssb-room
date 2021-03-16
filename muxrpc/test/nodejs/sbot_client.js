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

let createSbot = theStack({caps: {shs: testAppkey } })
  .use(require('ssb-db'))

const testName = process.env.TEST_NAME

// the other peer we are talking to
const testPeerAddr = process.env.TEST_PEERADDR
const testPeerRef = process.env.TEST_PEERREF

const testSession = require(process.env['TEST_SESSIONSCRIPT'])

// load the plugins needed for this session
for (plug of testSession.secretStackPlugins) {
  createSbot = createSbot.use(require(plug))
}

function bufFromEnv(evname) {
  const has = process.env[evname]
  if (has) {
    return Buffer.from(has, 'base64')
  }
  return false
}

tape.createStream().pipe(process.stderr)
tape(testName, function (t) {
  let timeoutLength = 30000
  var tapeTimeout = null
  function ready() { // needs to be called by the before block when it's done
    t.timeoutAfter(timeoutLength) // doesn't exit the process
    tapeTimeout = setTimeout(() => {
      t.comment('test timeout')
      process.exit(1)
    }, timeoutLength)
    const to = `net:${testPeerAddr}~shs:${testPeerRef.substr(1).replace('.ed25519', '')}`
    t.comment('dialing:' + to)
    sbot.conn.connect(to, (err, rpc) => {
      t.error(err, 'connected')
      t.comment('connected to: '+rpc.id)
      testSession.after(t, sbot, rpc, exit)
    })
  }

  function exit() { // call this when you're done
    sbot.close()
    t.comment('closed client: '+testName)
    clearTimeout(tapeTimeout)
    t.end()
    process.exit(0)
  }

  const tempRepo = process.env['TEST_REPO']
  console.warn(tempRepo)
  const keys = loadOrCreateSync(Path.join(tempRepo, 'secret'))
  const opts = {
    allowPrivate: true,
    path: tempRepo,
    keys: keys
  }

  opts.connections = {
    incoming: {
      tunnel: [{scope: 'public', transform: 'shs'}],
    },
    outgoing: {
      net: [{transform: 'shs'}],
      // ws: [{transform: 'shs'}],
      tunnel: [{transform: 'shs'}],
    },
  }


  if (testSHSappKey !== false) {
    opts.caps = opts.caps ? opts.caps : {}
    opts.caps.shs = testSHSappKey
  }

  const sbot = createSbot(opts)
  const alice = sbot.whoami()
  t.comment('client spawned. I am:' +  alice.id)
  
  console.log(alice.id) // tell go process who's incoming
  testSession.before(t, sbot, ready)
})
