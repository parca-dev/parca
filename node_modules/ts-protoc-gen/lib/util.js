"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
var querystring_1 = require("querystring");
var parameters_1 = require("./parameters");
function filePathToPseudoNamespace(filePath) {
    return filePath.replace(".proto", "").replace(/\//g, "_").replace(/\./g, "_").replace(/\-/g, "_") + "_pb";
}
exports.filePathToPseudoNamespace = filePathToPseudoNamespace;
function throwError(message, includeUnexpectedBehaviourMessage) {
    if (includeUnexpectedBehaviourMessage === void 0) { includeUnexpectedBehaviourMessage = true; }
    if (includeUnexpectedBehaviourMessage) {
        throw new Error(message + " - This is unhandled behaviour and should be reported as an issue on https://github.com/improbable-eng/ts-protoc-gen/issues");
    }
    else {
        throw new Error(message);
    }
}
exports.throwError = throwError;
function stripPrefix(str, prefix) {
    if (str.substr(0, prefix.length) === prefix) {
        return str.substr(prefix.length);
    }
    return str;
}
exports.stripPrefix = stripPrefix;
function snakeToCamel(str) {
    return str.replace(/(\_\w)/g, function (m) {
        return m[1].toUpperCase();
    });
}
exports.snakeToCamel = snakeToCamel;
function uppercaseFirst(str) {
    return str.charAt(0).toUpperCase() + str.slice(1);
}
exports.uppercaseFirst = uppercaseFirst;
var PROTO2_SYNTAX = "proto2";
function isProto2(fileDescriptor) {
    return (fileDescriptor.getSyntax() === "" || fileDescriptor.getSyntax() === PROTO2_SYNTAX);
}
exports.isProto2 = isProto2;
function oneOfName(name) {
    return uppercaseFirst(snakeToCamel(name.toLowerCase()));
}
exports.oneOfName = oneOfName;
function generateIndent(indentLevel) {
    var indent = "";
    for (var i = 0; i < indentLevel; i++) {
        indent += "  ";
    }
    return indent;
}
exports.generateIndent = generateIndent;
function getPathToRoot(fileName) {
    var depth = fileName.split("/").length;
    return depth === 1 ? "./" : new Array(depth).join("../");
}
exports.getPathToRoot = getPathToRoot;
function withinNamespaceFromExportEntry(name, exportEntry) {
    return exportEntry.pkg ? name.substring(exportEntry.pkg.length + 1) : name;
}
exports.withinNamespaceFromExportEntry = withinNamespaceFromExportEntry;
function replaceProtoSuffix(protoFilePath) {
    var suffix = ".proto";
    var hasProtoSuffix = protoFilePath.slice(protoFilePath.length - suffix.length) === suffix;
    return hasProtoSuffix
        ? protoFilePath.slice(0, -suffix.length) + "_pb"
        : protoFilePath;
}
exports.replaceProtoSuffix = replaceProtoSuffix;
function withAllStdIn(callback) {
    var ret = [];
    var len = 0;
    var stdin = process.stdin;
    stdin.on("readable", function () {
        var chunk;
        while ((chunk = stdin.read())) {
            if (!(chunk instanceof Buffer))
                throw new Error("Did not receive buffer");
            ret.push(chunk);
            len += chunk.length;
        }
    });
    stdin.on("end", function () {
        callback(Buffer.concat(ret, len));
    });
}
exports.withAllStdIn = withAllStdIn;
function normaliseFieldObjectName(name) {
    switch (name) {
        case "abstract":
        case "boolean":
        case "break":
        case "byte":
        case "case":
        case "catch":
        case "char":
        case "class":
        case "const":
        case "continue":
        case "debugger":
        case "default":
        case "delete":
        case "do":
        case "double":
        case "else":
        case "enum":
        case "export":
        case "extends":
        case "false":
        case "final":
        case "finally":
        case "float":
        case "for":
        case "function":
        case "goto":
        case "if":
        case "implements":
        case "import":
        case "in":
        case "instanceof":
        case "int":
        case "interface":
        case "long":
        case "native":
        case "new":
        case "null":
        case "package":
        case "private":
        case "protected":
        case "public":
        case "return":
        case "short":
        case "static":
        case "super":
        case "switch":
        case "synchronized":
        case "this":
        case "throw":
        case "throws":
        case "transient":
        case "try":
        case "typeof":
        case "var":
        case "void":
        case "volatile":
        case "while":
        case "with":
            return "pb_" + name;
    }
    return name;
}
exports.normaliseFieldObjectName = normaliseFieldObjectName;
function getServiceParameter(service) {
    switch (service) {
        case "true":
            console.warn("protoc-gen-ts warning: The service=true parameter has been deprecated. Use service=grpc-web instead.");
            return parameters_1.ServiceParameter.GrpcWeb;
        case "grpc-web":
            return parameters_1.ServiceParameter.GrpcWeb;
        case "grpc-node":
            return parameters_1.ServiceParameter.GrpcNode;
        case undefined:
            return parameters_1.ServiceParameter.None;
        default:
            throw new Error("Unrecognised service parameter: " + service);
    }
}
exports.getServiceParameter = getServiceParameter;
function getModeParameter(mode) {
    switch (mode) {
        case "grpc-js":
            return parameters_1.ModeParameter.GrpcJs;
        case undefined:
            return parameters_1.ModeParameter.None;
        default:
            throw new Error("Unrecognised mode parameter: " + mode);
    }
}
exports.getModeParameter = getModeParameter;
function getParameterEnums(parameter) {
    var _a = querystring_1.parse(parameter, ","), service = _a.service, mode = _a.mode;
    return {
        service: getServiceParameter(service),
        mode: getModeParameter(mode)
    };
}
exports.getParameterEnums = getParameterEnums;
//# sourceMappingURL=util.js.map