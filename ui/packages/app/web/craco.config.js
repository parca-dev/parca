const path = require('path');
const {getLoader, loaderByName} = require('@craco/craco');

const packages = [];
packages.push(path.join(__dirname, '../../shared/client'));
packages.push(path.join(__dirname, '../../shared/components'));
packages.push(path.join(__dirname, '../../shared/dynamicsize'));
packages.push(path.join(__dirname, '../../shared/functions'));
packages.push(path.join(__dirname, '../../shared/icons'));
packages.push(path.join(__dirname, '../../shared/parser'));
packages.push(path.join(__dirname, '../../shared/profile'));

module.exports = {
  webpack: {
    configure: (webpackConfig, arg) => {
      const {isFound, match} = getLoader(webpackConfig, loaderByName('babel-loader'));
      if (isFound) {
        const include = Array.isArray(match.loader.include)
          ? match.loader.include
          : [match.loader.include];

        match.loader.include = include.concat(packages);
      }
      return webpackConfig;
    },
  },
};
