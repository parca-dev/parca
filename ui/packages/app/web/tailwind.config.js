import typography from '@tailwindcss/typography';
import defaultConfig from 'tailwindcss/stubs/defaultConfig.stub.js';

import parcaComponentsConfig from '@parca/components/tailwind.config.js';
import parcaProfileConfig from '@parca/profile/tailwind.config.js';

const config = {
  presets: [parcaComponentsConfig, parcaProfileConfig],
  content: [
    './src/pages/**/*.{js,ts,jsx,tsx,mdx}',
    './src/components/**/*.{js,ts,jsx,tsx,mdx}',
    '../../shared/*/dist/**/*.{js,ts,jsx,tsx,mdx}',
  ],
  darkMode: 'class',
  theme: {
    extend: {
      fontFamily: {
        robotoMono: ['Roboto Mono', 'monospace'],
        sans: ['Poppins', ...defaultConfig.theme.fontFamily.sans],
      },
      maxWidth: {
        '1/2': '50%',
      },
      minWidth: theme => ({
        ...theme('spacing'),
      }),
      minHeight: theme => ({
        ...theme('spacing'),
      }),
      backgroundImage: {
        'shimmer-gradient':
          'linear-gradient(to right, transparent 0%, rgba(243, 243, 243, 0.8) 50%, transparent 100%)',
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
    },
  },
  variants: {},
  plugins: [typography],
};

export default config;
