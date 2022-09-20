const path = require('path');

module.exports = {
  reactOptions: {
    strictMode: true,
  },
  features: {
    storyStoreV7: true,
  },
  framework: '@storybook/react',
  core: {
    builder: 'webpack5',
  },
  stories: ['../packages/**/*.stories.mdx', '../packages/**/*.stories.@(js|jsx|ts|tsx)'],
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
            path: '../packages/app/web/postcss.config.js',
          },
        },
      },
    },
  ],
  // A workaround for a storybook regression, see https://github.com/storybookjs/storybook/issues/14197#issuecomment-949337652
  babel: async options => ({
    ...options,
    plugins: [['@babel/plugin-proposal-class-properties', {loose: true}]],
  }),
  webpackFinal: async (config, {configType}) => {
    // `configType` has a value of 'DEVELOPMENT' or 'PRODUCTION'
    // You can change the configuration based on that.
    // 'PRODUCTION' is used when building the static version of storybook.

    // Make whatever fine-grained changes you need
    config.module.rules[0].test = /\.(mjs|tsx?|ts?|jsx?)$/;
    config.module.rules.push({
      test: /\.scss$/,
      use: [
        {loader: 'style-loader'},
        {
          loader: 'css-loader',
          options: {modules: true},
        },
        {
          loader: 'postcss-loader',
          options: {
            postcssOptions: {
              plugins: ['tailwindcss', 'autoprefixer'],
            },
          },
        },
        {loader: 'sass-loader'},
      ],
      include: path.resolve(__dirname, '../'),
    });

    config.module.rules.push({
      test: /\.pb/,
      type: 'asset',
    });

    // Return the altered config
    return config;
  },
};
