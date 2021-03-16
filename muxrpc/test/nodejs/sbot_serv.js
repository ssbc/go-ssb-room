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

const testName = process.env['TEST_NAME']
const testPort = process.env['TEST_PORT']
const testSession = require(process.env['TEST_SESSIONSCRIPT'])

// load the plugins needed for this session
for (plug of testSession.secretStackPlugins) {
  createSbot = createSbot.use(require(plug))
}

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

  const tempRepo = process.env['TEST_REPO']
  console.warn("my repo:", tempRepo)
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
    t.comment('server spawned. I am:' +  alice.id)
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