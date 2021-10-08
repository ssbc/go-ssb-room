// SPDX-FileCopyrightText: 2021 The NGI Pointer Secure-Scuttlebutt Team of 2020/2021
//
// SPDX-License-Identifier: MIT

module.exports = {
  purge: ['../templates/**/*.tmpl'],
  darkMode: false,
  theme: {
    extend: {
      minHeight: (theme) => ({
        ...theme('spacing'),
      }),
    }
  },
  variants: {
    extend: {
      backgroundColor: ['odd'],
      zIndex: ['hover'],
    }
  },
  plugins: [],
};
