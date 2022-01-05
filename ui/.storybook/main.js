const path = require('path');

module.exports = {
  reactOptions: {
    strictMode: true,
  },
  core: {
    builder: 'webpack5',
  },
  stories: ['../packages/**/*.stories.mdx', '../packages/**/*.stories.@(js|jsx|ts|tsx)'],
  addons: ['@storybook/addon-links', '@storybook/addon-essentials'],
  webpackFinal: async (config, {configType}) => {
    // `configType` has a value of 'DEVELOPMENT' or 'PRODUCTION'
    // You can change the configuration based on that.
    // 'PRODUCTION' is used when building the static version of storybook.

    // Make whatever fine-grained changes you need
    config.module.rules.push({
      test: /\.scss$/,
      use: [
        {loader: 'style-loader'},
        {
          loader: 'css-loader',
          options: {modules: true},
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
