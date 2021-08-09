## 0.15.0

### Changes
* [#272](https://github.com/improbable-eng/ts-protoc-gen/pull/272) Fix get/set conflicting method names. ([@pkwarren](https://github.com/pkwarren)).
* [#275](https://github.com/improbable-eng/ts-protoc-gen/pull/275) Add support for proto3 optional presence. ([@awbraunstein](https://github.com/awbraunstein)).
* [#276](https://github.com/improbable-eng/ts-protoc-gen/pull/276) Fixed primitive extension handling. ([@marcuslongmuir](https://github.com/marcuslongmuir)).

## 0.14.0

### Changes
* [#247](https://github.com/improbable-eng/ts-protoc-gen/pull/247) Added support for grpc and @grpc/grpc-js server interfaces. ([@badsyntax](https://github.com/badsyntax)).

## 0.13.0

### Changes
* [#236](https://github.com/improbable-eng/ts-protoc-gen/pull/236) Added support for @grpc/grpc-js. ([@badsyntax](https://github.com/badsyntax)).

## 0.12.0

### Changes
* [#207](https://github.com/improbable-eng/ts-protoc-gen/pull/207) Bazel rules moved to a new repo, see README for migration guide. ([@Dig-Doug](https://github.com/Dig-Doug)).

## 0.11.0

### Changes
* [#185](https://github.com/improbable-eng/ts-protoc-gen/pull/185) Bazel rules add ES6 output ([@Dig-Doug](https://github.com/Dig-Doug))
* [#193](https://github.com/improbable-eng/ts-protoc-gen/pull/193) Add support for generating grpc-node service types ([@esilkensen](https://github.com/esilkensen))
* [#194](https://github.com/improbable-eng/ts-protoc-gen/pull/194) Bazel rules output UMD modules ([@Dig-Doug](https://github.com/Dig-Doug))

### Fixes
* [#183](https://github.com/improbable-eng/ts-protoc-gen/pull/183) Bugfix for field names with leading underscores ([@jonny-improbable](https://github.com/jonny-improbable))
* [#191](https://github.com/improbable-eng/ts-protoc-gen/pull/191) Bugfix for bazel rules where names with numbers were not being exported ([@Dig-Doug](https://github.com/Dig-Doug)) 

## 0.10.0

### Changes
* [#157](https://github.com/improbable-eng/ts-protoc-gen/pull/157) Generate more accurate types for Proto Enum values. ([@mattvagni](https://github.com/mattvagni))
* [#159](https://github.com/improbable-eng/ts-protoc-gen/pull/159) Swap ordering of `onStatus` and `onEnd` callbacks. ([@hectim](https://github.com/hectim))
* [#160](https://github.com/improbable-eng/ts-protoc-gen/pull/160) Update bazel-related library versions. ([@Dig-Doug](https://github.com/Dig-Doug))


### Fixes
* [#165](https://github.com/improbable-eng/ts-protoc-gen/pull/165) Replace uses of the deprecated `new Buffer()` with `Buffer.from()`. ([@ashi009](https://github.com/ashi009))
* [#161](https://github.com/improbable-eng/ts-protoc-gen/pull/161) Mark `google-protobuf` as a runtime dependency. ([@jonny-improbable](https://github.com/jonny-improbable))

## 0.9.0

### Changes
* [#147](https://github.com/improbable-eng/ts-protoc-gen/pull/147) Use `@improbable-eng/grpc-web` package instead of the soon to be deprecated `grpc-web-client` package. ([@johanbrandhorst](https://github.com/johanbrandhorst))

## 0.8.0

### Fixes
* [#131](https://github.com/improbable-eng/ts-protoc-gen/pull/131) Fix code-gen problems in client-side and bi-di stream stubs. ([@johanbrandhorst](https://github.com/johanbrandhorst))

### Changes
* [#139](https://github.com/improbable-eng/ts-protoc-gen/pull/139) Provide support for grpc-web-client v0.7.0+ ([@jonny-improbable](https://github.com/jonny-improbable))
* [#124](https://github.com/improbable-eng/ts-protoc-gen/pull/124) Provide support for cancelling unary calls. ([@virtuald](https://github.com/virtuald))

## 0.7.7

### Fixes
* Replace usage of `Object.assign` to fix webpack issue. [@jonny-improbable](https://github.com/jonny-improbable) in [#110](https://github.com/improbable-eng/ts-protoc-gen/pull/110)
* Errors returned by Unary Services should be optionally null. [@colinking](https://github.com/collinking) in [#116](https://github.com/improbable-eng/ts-protoc-gen/pull/116)
* Fix snake_cased oneof message are generated to incorrect types. [@riku179](https://github.com/riku179) in [#118](https://github.com/improbable-eng/ts-protoc-gen/pull/118)
* `.deb` artificats being deployment to npm. [@jonnyreeves](https://github.com/jonnyreeves) in [#121](https://github.com/improbable-eng/ts-protoc-gen/pull/121)

### Changes
* Add support for `jstype` proto annotations. [@jonny-improbable](https://github.com/jonny-improbable) in [#104](https://github.com/improbable-eng/ts-protoc-gen/pull/104)
* Implement Client Streaming and BiDi Streaming for grpc-web service stubs. [@jonnyreeves](https://github.com/jonnyreeves) in [#82](https://github.com/improbable-eng/ts-protoc-gen/pull/82)

## 0.7.6

### Fixes
* Broken integration tests on master 

## 0.7.5

### Fixes
* Fixed NPM publish.

## 0.7.4

### Changes
* Download protoc when generating protos to ensure a consistent version is being used. [@easyCZ](https://github.com/easyCZ) in [#80](https://github.com/improbable-eng/ts-protoc-gen/pull/80) 
* Always generate Service Definitions (`pb_service.d.js` and `pb_service.d.ts`) even if the proto does not define any services. [@lx223 ](https://github.com/lx223) in [#83](https://github.com/improbable-eng/ts-protoc-gen/pull/83) 
* Add custom Bazel rule which uses ts-protoc-gen for generation. [@coltonmorris](https://github.com/coltonmorris) in [#84](https://github.com/improbable-eng/ts-protoc-gen/pull/84)
* Add `debug` to `ServiceClientOptions`. [@bianbian-org](https://github.com/bianbian-org) in [#90](https://github.com/improbable-eng/ts-protoc-gen/pull/90)

## 0.7.3

### Changes
* None (testing release script...)

## 0.7.1

### Changes
* Fixing bad npm publish

## 0.7.0

### Changes
* Don't use reserved keywords as function names in grpc service stubs [@jonahbron](https://github.com/jonahbron) and [@jonny-improbable]((https://github.com/jonny-improbable)) in [#61](https://github.com/improbable-eng/ts-protoc-gen/pull/61)

### Fixes
* Fix casing mismatch for oneOf declarations. [@jonnyreeves](https://github.com/jonnyreeves) in [#67](https://github.com/improbable-eng/ts-protoc-gen/pull/67)
* Fix Bazel build [@coltonmorris](https://github.com/coltonmorris) in [#71](https://github.com/improbable-eng/ts-protoc-gen/pull/71)

## 0.6.0

### Changes
* Generate gRPC Service Stubs for use with grpc-web [@jonahbron](https://github.com/jonahbron) and [@jonny-improbable](https://github.com/jonny-improbable) in [#40](https://github.com/improbable-eng/ts-protoc-gen/pull/40)
* Fix filename manipulation bug which would cause problems for users who store generated files with `.proto` in the path. [@easyCZ](https://github.com/easyCZ) in [#56](https://github.com/improbable-eng/ts-protoc-gen/pull/56)

## 0.5.2

### Changes
* Fixes invalid 0.5.1 publish (fixed prepublishOnly script)

## 0.5.1

### Changes
* Fixes invalid 0.5.0 publish (added prepublishOnly script)

## 0.5.0

### Migration Guide
The `protoc-gen-js_service` command has been removed as the `protoc-gen-ts` command now generates both JavaScript and TypeScript. Consumers of `protoc-gen-js_service` should instead use `protoc-gen-ts` and substitute the `--js_service_out=generated` protoc flag with `--ts_out=service=true:generated`.

### Changes
* Export Enum Definitions as ALL_CAPS [@jonnyreeves](https://github.com/jonnyreeves) in [#22](https://github.com/improbable-eng/ts-protoc-gen/issues/22)
* Don't output variables that are not used in typescript service definition [@jonbretman](https://github.com/jonbretman) in [#38](https://github.com/improbable-eng/ts-protoc-gen/pull/38)
* Support Bazel build [@adamyi](https://github.com/adamyi) in [#34](https://github.com/improbable-eng/ts-protoc-gen/pull/34)
* Create JavaScript sources and TypeScript definitions for grpc-web services [@jonny-improbable](https://github.com/jonny-improbable) in [#44](https://github.com/improbable-eng/ts-protoc-gen/pull/44)
* Stop using TypeScript Modules in generated grpc-web Service Definitions [@jonny-improbable](https://github.com/jonny-improbable) in [#45](https://github.com/improbable-eng/ts-protoc-gen/pull/45)

## 0.4.0

### Changes
*  Add `pb_` prefix to JS Reserved Keywords [@jonnyreeves](https://github.com/jonnyreeves) in [#20](https://github.com/improbable-eng/ts-protoc-gen/pull/20)

## 0.3.3

### Changes
* Fix error on messages without packages [@MarcusLongmuir](https://github.com/MarcusLongmuir) in [#13](https://github.com/improbable-eng/ts-protoc-gen/pull/13) 
