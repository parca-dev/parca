// Shared Babel config for packages that need the React compiler.
// Used by shared packages to compile React components with the compiler plugin.
module.exports = {
  ignore: [
    '**/*.test.ts',
    '**/*.test.tsx',
    '**/*.benchmark.ts',
    '**/*.benchmark.tsx',
    '**/benchdata/**',
    '**/testdata/**',
    '**/.DS_Store',
    '**/*.md',
  ],
  presets: [
    ['@babel/preset-env', {modules: false}],
    ['@babel/preset-react', {runtime: 'automatic'}],
    '@babel/preset-typescript',
  ],
  plugins: [
    [
      'babel-plugin-react-compiler',
      {
        target: '18',
        compilationMode: 'infer',
      },
    ],
  ],
};
