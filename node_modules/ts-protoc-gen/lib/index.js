"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
var fileDescriptorTSD_1 = require("./ts/fileDescriptorTSD");
var ExportMap_1 = require("./ExportMap");
var util_1 = require("./util");
var plugin_pb_1 = require("google-protobuf/google/protobuf/compiler/plugin_pb");
var grpcweb_1 = require("./service/grpcweb");
var grpcnode_1 = require("./service/grpcnode");
var parameters_1 = require("./parameters");
util_1.withAllStdIn(function (inputBuff) {
    try {
        var typedInputBuff = new Uint8Array(inputBuff.length);
        typedInputBuff.set(inputBuff);
        var codeGenRequest = plugin_pb_1.CodeGeneratorRequest.deserializeBinary(typedInputBuff);
        var codeGenResponse_1 = new plugin_pb_1.CodeGeneratorResponse();
        codeGenResponse_1.setSupportedFeatures(plugin_pb_1.CodeGeneratorResponse.Feature.FEATURE_PROTO3_OPTIONAL);
        var exportMap_1 = new ExportMap_1.ExportMap();
        var fileNameToDescriptor_1 = {};
        var parameter = codeGenRequest.getParameter();
        var _a = util_1.getParameterEnums(parameter || ""), service = _a.service, mode_1 = _a.mode;
        var generateGrpcWebServices_1 = service === parameters_1.ServiceParameter.GrpcWeb;
        var generateGrpcNodeServices_1 = service === parameters_1.ServiceParameter.GrpcNode;
        codeGenRequest.getProtoFileList().forEach(function (protoFileDescriptor) {
            var fileDescriptorName = protoFileDescriptor.getName() || util_1.throwError("Missing file descriptor name");
            fileNameToDescriptor_1[fileDescriptorName] = protoFileDescriptor;
            exportMap_1.addFileDescriptor(protoFileDescriptor);
        });
        codeGenRequest.getFileToGenerateList().forEach(function (fileName) {
            var outputFileName = util_1.replaceProtoSuffix(fileName);
            var thisFile = new plugin_pb_1.CodeGeneratorResponse.File();
            thisFile.setName(outputFileName + ".d.ts");
            thisFile.setContent(fileDescriptorTSD_1.printFileDescriptorTSD(fileNameToDescriptor_1[fileName], exportMap_1));
            codeGenResponse_1.addFile(thisFile);
            if (generateGrpcWebServices_1) {
                grpcweb_1.generateGrpcWebService(outputFileName, fileNameToDescriptor_1[fileName], exportMap_1)
                    .forEach(function (file) { return codeGenResponse_1.addFile(file); });
            }
            else if (generateGrpcNodeServices_1) {
                var file = grpcnode_1.generateGrpcNodeService(outputFileName, fileNameToDescriptor_1[fileName], exportMap_1, mode_1);
                codeGenResponse_1.addFile(file);
            }
        });
        process.stdout.write(Buffer.from(codeGenResponse_1.serializeBinary().buffer));
    }
    catch (err) {
        console.error("protoc-gen-ts error: " + err.stack + "\n");
        process.exit(1);
    }
});
//# sourceMappingURL=index.js.map