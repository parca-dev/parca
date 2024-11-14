module.exports = {
  content: ['./src/**/*.{html,js,jsx,ts,tsx}'],
  darkMode: 'class',
  theme: {
    extend: {
      backgroundImage: {
        'shimmer-gradient':
          'linear-gradient(to right, transparent 0%, rgba(255, 255, 255, 0.8) 50%, transparent 100%)',
        'shimmer-gradient-dark':
          'linear-gradient(to right, transparent 0%, rgb(17 24 39 / 26%) 50%, transparent 100%)',
      },
      animation: {
        shimmer: 'shimmer 2s infinite linear',
      },
      keyframes: {
        shimmer: {
          '0%': {transform: 'translateX(-100%)'},
          '100%': {transform: 'translateX(100%)'},
        },
      },
      maxWidth: theme => ({
        ...theme('spacing'),
      }),
      minWidth: theme => ({
        ...theme('spacing'),
      }),
      maxHeight: theme => ({
        ...theme('spacing'),
      }),
      minHeight: theme => ({
        ...theme('spacing'),
      }),
    },
  },
  plugins: [],
};
