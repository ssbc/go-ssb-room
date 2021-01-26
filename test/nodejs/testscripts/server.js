module.exports = {
    before: (sbot, ready) => {
        console.warn('before:', sbot.id)
        setTimeout(ready, 1000)
        // ready()
    },

    after: (sbot, client) => {
        console.warn('after:', sbot.id, client.id)
    }
}