# Contributing

Firstly, thanks for taking the time to contribute.

The following is a set of guidelines for contributing to `ts-protoc-gen`.

## Pull Request Process
1. Implement your changes. If you're unsure how or would like clarification, please raise an [issue](https://github.com/improbable-eng/ts-protoc-gen/issues/new).
2. Ensure your change has sufficient test coverage.
3. Ensure all changes pass the lint check.
2. Maintainers will look to review the PR as soon as possible. If there is no traction for some time, you're welcome to bump the thread.
3. All PRs require at least one reviewer.

## Code of Conduct
Please review and follow our [Code of Conduct](https://github.com/improbable-eng/ts-protoc-gen/blob/master/CODE_OF_CONDUCT.md).

## Releasing
Your changes will be released with the next version release.

## Debugging
You can attach the Chrome Inspector when running the tests by setting the `MOCHA_DEBUG` environment variable before running the tests, ie:

```
MOCHA_DEBUG=true npm test
```
