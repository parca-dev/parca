"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
var Printer_1 = require("../Printer");
var common_1 = require("./common");
var parameters_1 = require("../parameters");
function generateGrpcNodeService(filename, descriptor, exportMap, modeParameter) {
    var definitionFilename = filename.replace(/_pb$/, "_grpc_pb.d.ts");
    return common_1.createFile(generateTypeScriptDefinition(descriptor, exportMap, modeParameter), definitionFilename);
}
exports.generateGrpcNodeService = generateGrpcNodeService;
function generateTypeScriptDefinition(fileDescriptor, exportMap, modeParameter) {
    var serviceDescriptor = new common_1.GrpcServiceDescriptor(fileDescriptor, exportMap);
    var printer = new Printer_1.Printer(0);
    var hasServices = serviceDescriptor.services.length > 0;
    if (hasServices) {
        printer.printLn("// GENERATED CODE -- DO NOT EDIT!");
        printer.printEmptyLn();
    }
    else {
        printer.printLn("// GENERATED CODE -- NO SERVICES IN PROTO");
        return printer.getOutput();
    }
    printer.printLn("// package: " + serviceDescriptor.packageName);
    printer.printLn("// file: " + serviceDescriptor.filename);
    printer.printEmptyLn();
    serviceDescriptor.imports
        .forEach(function (importDescriptor) {
        printer.printLn("import * as " + importDescriptor.namespace + " from \"" + importDescriptor.path + "\";");
    });
    var importPackage = modeParameter === parameters_1.ModeParameter.GrpcJs ? "@grpc/grpc-js" : "grpc";
    printer.printLn("import * as grpc from \"" + importPackage + "\";");
    serviceDescriptor.services
        .forEach(function (service) {
        printer.printEmptyLn();
        printService(printer, service);
        printer.printEmptyLn();
        printServer(printer, service);
        printer.printEmptyLn();
        printClient(printer, service);
    });
    return printer.getOutput();
}
function printService(printer, service) {
    var serviceName = service.name + "Service";
    printer.printLn("interface I" + serviceName + " extends grpc.ServiceDefinition<grpc.UntypedServiceImplementation> {");
    service.methods
        .forEach(function (method) {
        var methodType = "grpc.MethodDefinition<" + method.requestType + ", " + method.responseType + ">";
        printer.printIndentedLn(method.nameAsCamelCase + ": " + methodType + ";");
    });
    printer.printLn("}");
    printer.printEmptyLn();
    printer.printLn("export const " + serviceName + ": I" + serviceName + ";");
}
function printServer(printer, service) {
    var serverName = service.name + "Server";
    printer.printLn("export interface I" + serverName + " extends grpc.UntypedServiceImplementation {");
    service.methods
        .forEach(function (method) {
        if (!method.requestStream && !method.responseStream) {
            printer.printIndentedLn(method.nameAsCamelCase + ": grpc.handleUnaryCall<" + method.requestType + ", " + method.responseType + ">;");
        }
        else if (!method.requestStream) {
            printer.printIndentedLn(method.nameAsCamelCase + ": grpc.handleServerStreamingCall<" + method.requestType + ", " + method.responseType + ">;");
        }
        else if (!method.responseStream) {
            printer.printIndentedLn(method.nameAsCamelCase + ": grpc.handleClientStreamingCall<" + method.requestType + ", " + method.responseType + ">;");
        }
        else {
            printer.printIndentedLn(method.nameAsCamelCase + ": grpc.handleBidiStreamingCall<" + method.requestType + ", " + method.responseType + ">;");
        }
    });
    printer.printLn("}");
}
function printClient(printer, service) {
    printer.printLn("export class " + service.name + "Client extends grpc.Client {");
    printer.printIndentedLn("constructor(address: string, credentials: grpc.ChannelCredentials, options?: object);");
    service.methods
        .forEach(function (method) {
        if (!method.requestStream && !method.responseStream) {
            printUnaryRequestMethod(printer, method);
        }
        else if (!method.requestStream) {
            printServerStreamRequestMethod(printer, method);
        }
        else if (!method.responseStream) {
            printClientStreamRequestMethod(printer, method);
        }
        else {
            printBidiStreamRequest(printer, method);
        }
    });
    printer.printLn("}");
}
var metadata = "metadata: grpc.Metadata | null";
var options = "options: grpc.CallOptions | null";
var metadataOrOptions = "metadataOrOptions: grpc.Metadata | grpc.CallOptions | null";
var optionalMetadata = "metadata?: grpc.Metadata | null";
var optionalOptions = "options?: grpc.CallOptions | null";
var optionalMetadataOrOptions = "metadataOrOptions?: grpc.Metadata | grpc.CallOptions | null";
function printUnaryRequestMethod(printer, method) {
    var name = method.nameAsCamelCase;
    var argument = "argument: " + method.requestType;
    var callback = "callback: grpc.requestCallback<" + method.responseType + ">";
    var returnType = "grpc.ClientUnaryCall";
    printer.printIndentedLn(name + "(" + argument + ", " + callback + "): " + returnType + ";");
    printer.printIndentedLn(name + "(" + argument + ", " + metadataOrOptions + ", " + callback + "): " + returnType + ";");
    printer.printIndentedLn(name + "(" + argument + ", " + metadata + ", " + options + ", " + callback + "): " + returnType + ";");
}
function printServerStreamRequestMethod(printer, method) {
    var name = method.nameAsCamelCase;
    var argument = "argument: " + method.requestType;
    var returnType = "grpc.ClientReadableStream<" + method.responseType + ">";
    printer.printIndentedLn(name + "(" + argument + ", " + optionalMetadataOrOptions + "): " + returnType + ";");
    printer.printIndentedLn(name + "(" + argument + ", " + optionalMetadata + ", " + optionalOptions + "): " + returnType + ";");
}
function printClientStreamRequestMethod(printer, method) {
    var name = method.nameAsCamelCase;
    var callback = "callback: grpc.requestCallback<" + method.responseType + ">";
    var returnType = "grpc.ClientWritableStream<" + method.requestType + ">";
    printer.printIndentedLn(name + "(" + callback + "): grpc.ClientWritableStream<" + method.requestType + ">;");
    printer.printIndentedLn(name + "(" + metadataOrOptions + ", " + callback + "): " + returnType + ";");
    printer.printIndentedLn(name + "(" + metadata + ", " + options + ", " + callback + "): " + returnType + ";");
}
function printBidiStreamRequest(printer, method) {
    var name = method.nameAsCamelCase;
    var returnType = "grpc.ClientDuplexStream<" + method.requestType + ", " + method.responseType + ">";
    printer.printIndentedLn(name + "(" + optionalMetadataOrOptions + "): " + returnType + ";");
    printer.printIndentedLn(name + "(" + optionalMetadata + ", " + optionalOptions + "): " + returnType + ";");
}
//# sourceMappingURL=grpcnode.js.map