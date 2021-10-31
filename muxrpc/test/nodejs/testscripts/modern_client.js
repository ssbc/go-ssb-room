// SPDX-FileCopyrightText: 2021 The NGI Pointer Secure-Scuttlebutt Team of 2020/2021
//
// SPDX-License-Identifier: MIT

const secretStackPlugins = require('./secretstack-modern')
const before = require('./minimal-before-setup')
const performClientTest = require('./client')
module.exports = {
    secretStackPlugins,
    before,
    after: performClientTest
}
