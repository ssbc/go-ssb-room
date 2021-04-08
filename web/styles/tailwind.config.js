module.exports = {
  purge: ['../templates/**/*.tmpl'],
  darkMode: false,
  theme: {
    extend: {},
  },
  variants: {
    extend: {
      backgroundColor: ['odd'],
      zIndex: ['hover'],
    }
  },
  plugins: [],
};
