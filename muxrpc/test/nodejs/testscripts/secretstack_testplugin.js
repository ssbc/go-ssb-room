// SPDX-FileCopyrightText: 2021 The NGI Pointer Secure-Scuttlebutt Team of 2020/2021
//
// SPDX-License-Identifier: MIT

/*
    this testing plugin supplies a very simple method to see if the other side is working
*/
module.exports = {
    name: 'testing',
    version: '1.0.0',
    manifest: {
        working: 'async'
    },
    permissions: {
        anonymous: { allow: ['working'] },
    },
    init(ssb) {
        return {
            working(cb) {
                cb(null, true)
            }
        };
    },
};