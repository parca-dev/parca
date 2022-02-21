const path = require('path');

module.exports = {
  stories: ['../src/**/*.stories.mdx', '../src/**/*.stories.@(js|jsx|ts|tsx)'],
  addons: [
    '@storybook/addon-docs',
    '@storybook/addon-links',
    '@storybook/addon-essentials',
    'storybook-dark-mode',
    {
      name: '@storybook/addon-postcss',
      options: {
        postcssLoaderOptions: {
          implementation: require('postcss'),
          postcssOptions: {
            path: './storybook/postcss.config.js',
          },
        },
      },
    },
  ],
  features: {
    storyStoreV7: true,
  },
  framework: '@storybook/react',
  core: {
    builder: 'webpack5',
  },
  // A workaround for a storybook regression, see https://github.com/storybookjs/storybook/issues/14197#issuecomment-949337652
  babel: async options => ({
    ...options,
    plugins: [['@babel/plugin-proposal-class-properties', {loose: true}]],
  }),
  webpackFinal: config => {
    config.module.rules[0].test = /\.(mjs|tsx?|ts?|jsx?)$/;
    config.module.rules.push({
      test: /\.scss$/,
      use: [
        'style-loader',
        'css-loader',
        {
          loader: 'postcss-loader',
          options: {
            postcssOptions: {
              plugins: ['tailwindcss', 'autoprefixer'],
            },
          },
        },
        'sass-loader',
      ],
    });

    // @see https://github.com/storybookjs/storybook/issues/11989#issuecomment-715524391
    config.resolve.alias = {
      ...config.resolve?.alias,
      '@': [path.resolve(__dirname, '../src/'), path.resolve(__dirname, '../')],
      components: path.resolve(__dirname, '../src/components/'),
      libs: path.resolve(__dirname, '../src/libs/'),
    };
    return config;
  },
};
