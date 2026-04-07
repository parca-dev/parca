// Shared Babel config for packages that need the React compiler.
// Used by shared packages to compile React components with the compiler plugin.
module.exports = {
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
