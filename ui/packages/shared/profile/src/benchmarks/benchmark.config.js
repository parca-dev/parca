module.exports = {
  rootDir: process.cwd(),
  moduleFileExtensions: ['js', 'jsx', 'ts', 'tsx', 'json', 'png', 'md', 'html'],
  verbose: true,
  testMatch: ['**/*.benchmark.[jt]s?(x)'],
  testPathIgnorePatterns: ['<rootDir>/node_modules/', '.(?:skip).'],
  modulePaths: ['.'],
  transform: {
    '\\.[jt]sx?$': [
      '@swc/jest',
      {
        rootMode: 'upward',
        jsc: {
          transform: {
            react: {
              runtime: 'automatic',
            },
          },
        },
      },
    ],
    '^.+\\.mdx$': '@storybook/addon-docs/jest-transform-mdx',
    '^.+\\.svg$': 'jest-transformer-svg',
  },
  transformIgnorePatterns: [],
  coveragePathIgnorePatterns: ['node_modules', 'out', '.next', '.storybook', '.stories.mdx'],
  moduleNameMapper: {
    '\\.(css|scss)$': 'identity-obj-proxy',
  },
  testEnvironment: 'jsdom',
  env: {
    'jest/globals': true,
  },
  testTimeout: 15000,
};
