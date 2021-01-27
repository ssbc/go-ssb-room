const Path = require('path')
const tape = require('tape')
const { loadOrCreateSync } = require('ssb-keys')
const theStack = require('secret-stack')
const ssbCaps = require('ssb-caps')

const testSHSappKey = bufFromEnv('TEST_APPKEY')

let testAppkey = Buffer.from(ssbCaps.shs, 'base64')
if (testSHSappKey !== false) {
  testAppkey = testSHSappKey
}

// stackOpts = {appKey: require('ssb-caps').shs}
stackOpts = {caps: {shs: testAppkey } }
const createSbot = theStack(stackOpts)
  .use(require('ssb-db'))  
  .use(require('ssb-master'))
  .use(require('ssb-logging'))
  .use(require('ssb-conn'))
  .use(require('ssb-room/tunnel/server'))

const testName = process.env['TEST_NAME']
const testPort = process.env['TEST_PORT']
const testSession = require(process.env['TEST_SESSIONSCRIPT'])

tape.createStream().pipe(process.stderr);
tape(testName, function (t) {
  // t.timeoutAfter(30000) // doesn't exit the process
//   const tapeTimeout = setTimeout(() => {
//     t.comment("test timeout")
//     process.exit(1)
//   }, 50000)
  
  

  function exit() { // call this when you're done
    sbot.close()
    t.comment('closed jsbot')
    // clearTimeout(tapeTimeout)
    t.end()
  }

  const tempRepo = Path.join('testrun', testName)
  const keys = loadOrCreateSync(Path.join(tempRepo, 'secret'))
  const sbot = createSbot({
    port: testPort,
    path: tempRepo,
    keys: keys,
  })
  const alice = sbot.whoami()

//   const replicate_changes = sbot.replicate.changes()

  t.comment("sbot spawned, running before")
 
  function ready() {
    console.warn('ready!', alice.id)
    console.log(alice.id) // tell go process who our pubkey
  }
  testSession.before(sbot, ready)
  
  
  sbot.on("rpc:connect", (remote, isClient) => {
    t.comment("new connection: "+ remote.id)
    testSession.after(sbot, remote, exit)
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