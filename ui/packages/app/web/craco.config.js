const path = require('path');
const {loaderByName, addAfterLoader, removeLoaders} = require('@craco/craco');

const packages = [__dirname];
packages.push(path.join(__dirname, '../../shared/client'));
packages.push(path.join(__dirname, '../../shared/components'));
packages.push(path.join(__dirname, '../../shared/dynamicsize'));
packages.push(path.join(__dirname, '../../shared/functions'));
packages.push(path.join(__dirname, '../../shared/icons'));
packages.push(path.join(__dirname, '../../shared/parser'));
packages.push(path.join(__dirname, '../../shared/profile'));
packages.push(path.join(__dirname, '../../shared/store'));
packages.push(path.join(__dirname, '../../shared/hooks'));
packages.push(path.join(__dirname, '../../shared/utilities'));

module.exports = {
  webpack: {
    configure: (webpackConfig, arg) => {
      addAfterLoader(webpackConfig, loaderByName('babel-loader'), {
        test: /\.(js|mjs|jsx|ts|tsx)$/,
        include: packages,
        loader: require.resolve('swc-loader'),
      });

      removeLoaders(webpackConfig, loaderByName('babel-loader'));

      return webpackConfig;
    },
  },
};
