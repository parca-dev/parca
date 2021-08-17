module.exports = {
  rootDir: process.cwd(),
  moduleFileExtensions: ['js', 'jsx', 'ts', 'tsx', 'json', 'png', 'md', 'html'],
  verbose: true,
  testPathIgnorePatterns: ['<rootDir>/node_modules/', '.(?:skip).'],
  modulePaths: ['.'],
  transform: {
    '\\.[jt]sx?$': [
      'babel-jest',
      {
        rootMode: 'upward'
      }
    ],
    '^.+\\.mdx$': '@storybook/addon-docs/jest-transform-mdx'
  },
  collectCoverage: true,
  coveragePathIgnorePatterns: ['node_modules', 'out', '.next', '.storybook', '.stories.mdx'],
  moduleNameMapper: {
    '\\.(css|scss)$': 'identity-obj-proxy'
  }
}

