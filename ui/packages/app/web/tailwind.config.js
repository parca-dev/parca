module.exports = {
  content: [
    './src/pages/**/*.{js,ts,jsx,tsx,mdx}',
    './src/components/**/*.{js,ts,jsx,tsx,mdx}',
    '../../shared/**/*.{js,ts,jsx,tsx,mdx}',
  ],
  darkMode: 'class',
  theme: {
    extend: {
      fontFamily: {
        robotoMono: ['Roboto Mono', 'monospace'],
      },
      maxWidth: {
        '1/2': '50%',
      },
      minWidth: theme => ({
        ...theme('spacing'),
      }),
    },
  },
  variants: {},
  plugins: [],
};
