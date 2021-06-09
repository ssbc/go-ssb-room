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
