const {dependencies} = require('./package.json');

const withTM = require('next-transpile-modules')(
  Object.keys(dependencies || []).filter(dependency => dependency.startsWith('@parca/'))
);

const withBundleAnalyzer = require('@next/bundle-analyzer')({
  enabled: process.env.ANALYZE === 'true',
});

module.exports = withBundleAnalyzer(
  withTM({
    trailingSlash: process.env.NODE_ENV === 'production',
    env: {
      NEXT_PUBLIC_BUILD_REVISION: process.env.BUILD_REVISION || 'DEVELOP',
    },
    basePath: process.env.PATH_PREFIX,
    webpack: config => {
      config.module.rules.push({
        test: /\.svg$/,
        use: ['@svgr/webpack'],
      });

      return config;
    },
  })
);
