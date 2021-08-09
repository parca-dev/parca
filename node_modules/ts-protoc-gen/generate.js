const { spawnSync } = require("child_process");

const { existsSync, mkdirSync } = require("fs");
const { resolve } = require("path");

const protocVersion = "3.15.5";

const examplesGeneratedPath = resolve(__dirname, "examples", "generated");
const examplesGeneratedGrpcWebPath = resolve(__dirname, "examples", "generated-grpc-web");
const examplesGeneratedGrpcNodePath = resolve(__dirname, "examples", "generated-grpc-node");
const examplesGeneratedGrpcJsNodePath = resolve(__dirname, "examples", "generated-grpc-js-node");

const binSuffix = process.platform === "win32" ? ".cmd" : "";
const nodeModulesBin = resolve(__dirname, "node_modules", ".bin");

const downloadPath = resolve(nodeModulesBin, "download") + binSuffix;

const protocRoot = resolve(__dirname, "protoc");
const protocPath = resolve(protocRoot, "bin", "protoc");

const protocPluginPath = resolve(__dirname, "bin", "protoc-gen-ts") + binSuffix;

const rimrafPath = resolve(nodeModulesBin, "rimraf") + binSuffix;

const supportedPlatforms = {
  darwin: {
    downloadSuffix: "osx-x86_64",
    name: "Mac"
  },
  linux: {
    downloadSuffix: "linux-x86_64",
    name: "Linux"
  },
  win32: {
    downloadSuffix: "win32",
    name: "Windows"
  }
};

const platform = supportedPlatforms[process.platform];
const platformName = platform ?
  platform.name :
  `UNKNOWN:${process.platform}`;
console.log("You appear to be running on", platformName);

requireBuild();

const glob = require("glob");

requireProtoc();

requireDir(examplesGeneratedPath);
requireDir(examplesGeneratedGrpcWebPath);
requireDir(examplesGeneratedGrpcNodePath);
requireDir(examplesGeneratedGrpcJsNodePath);

// Generate no services

run(protocPath,
  `--proto_path=${__dirname}`,
  `--plugin=protoc-gen-ts=${protocPluginPath}`,
  `--js_out=import_style=commonjs,binary:${examplesGeneratedPath}`,
  `--ts_out=${examplesGeneratedPath}`,
  ...glob.sync(resolve(__dirname, "proto", "**/*.proto"))
);

// Generate grpc-web services

run(protocPath,
  `--proto_path=${__dirname}`,
  `--plugin=protoc-gen-ts=${protocPluginPath}`,
  `--js_out=import_style=commonjs,binary:${examplesGeneratedGrpcWebPath}`,
  `--ts_out=service=grpc-web:${examplesGeneratedGrpcWebPath}`,
  ...glob.sync(resolve(__dirname, "proto", "**/*.proto"))
);

// Generate grpc-node services

run(protocPath,
  `--proto_path=${__dirname}`,
  `--plugin=protoc-gen-ts=${protocPluginPath}`,
  `--plugin=protoc-gen-grpc=node_modules/.bin/grpc_tools_node_protoc_plugin`,
  `--js_out=import_style=commonjs,binary:${examplesGeneratedGrpcNodePath}`,
  `--ts_out=service=grpc-node:${examplesGeneratedGrpcNodePath}`,
  `--grpc_out=${examplesGeneratedGrpcNodePath}`,
  ...glob.sync(resolve(__dirname, "proto", "**/*.proto"))
);

// Generate grpc-node services using grpc-js mode

run(protocPath,
  `--proto_path=${__dirname}`,
  `--plugin=protoc-gen-ts=${protocPluginPath}`,
  `--plugin=protoc-gen-grpc=node_modules/.bin/grpc_tools_node_protoc_plugin`,
  `--js_out=import_style=commonjs,binary:${examplesGeneratedGrpcJsNodePath}`,
  `--ts_out=service=grpc-node,mode=grpc-js:${examplesGeneratedGrpcJsNodePath}`,
  `--grpc_out=grpc_js:${examplesGeneratedGrpcJsNodePath}`,
  ...glob.sync(resolve(__dirname, "proto", "**/*.proto"))
);

run(rimrafPath, protocRoot);

function requireBuild() {
  console.log("Ensuring we have NPM packages installed...");
  run("npm", "install");

  console.log("Compiling ts-protoc-gen...");
  run("npm", "run", "build");
}

function requireProtoc() {
  if (existsSync(protocPath)) {
    return;
  }

  if (!platform) {
    throw new Error(
      "Cannot download protoc. " +
      platformName +
      " is not currently supported by ts-protoc-gen"
    );
  }

  console.log(`Downloading protoc v${protocVersion} for ${platform.name}`);
  const protocUrl =
    `https://github.com/google/protobuf/releases/download/v${protocVersion}/protoc-${protocVersion}-${platform.downloadSuffix}.zip`;

  run(downloadPath,
    "--extract",
    "--out", protocRoot,
    protocUrl);
}

function requireDir(path) {
  if (existsSync(path)) {
    run(rimrafPath, path);
  }

  mkdirSync(path);
}

function run(executablePath, ...args) {
  const result = spawnSync(executablePath, args, { shell: true, stdio: "inherit" });
  if (result.status !== 0) {
    throw new Error(`Exited ${executablePath} with status ${result.status}`);
  }
}
