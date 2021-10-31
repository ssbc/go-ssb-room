// SPDX-FileCopyrightText: 2021 The NGI Pointer Secure-Scuttlebutt Team of 2020/2021
//
// SPDX-License-Identifier: MIT

const secretStackPlugins = require('./secretstack-legacy')
const before = require('./minimal-before-setup')
const performOpeningTunnelTest = require('./client-opening-tunnel')

module.exports = {
    secretStackPlugins,
    before,
    after: performOpeningTunnelTest
}
