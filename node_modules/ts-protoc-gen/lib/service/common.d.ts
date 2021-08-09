import { CodeGeneratorResponse } from "google-protobuf/google/protobuf/compiler/plugin_pb";
import { FileDescriptorProto, ServiceDescriptorProto } from "google-protobuf/google/protobuf/descriptor_pb";
import { ExportMap } from "../ExportMap";
export declare function createFile(output: string, filename: string): CodeGeneratorResponse.File;
export declare type ImportDescriptor = {
    readonly namespace: string;
    readonly path: string;
};
export declare type RPCMethodDescriptor = {
    readonly nameAsPascalCase: string;
    readonly nameAsCamelCase: string;
    readonly functionName: string;
    readonly serviceName: string;
    readonly requestStream: boolean;
    readonly responseStream: boolean;
    readonly requestType: string;
    readonly responseType: string;
};
export declare class RPCDescriptor {
    private readonly grpcService;
    private readonly protoService;
    private readonly exportMap;
    constructor(grpcService: GrpcServiceDescriptor, protoService: ServiceDescriptorProto, exportMap: ExportMap);
    readonly name: string;
    readonly qualifiedName: string;
    readonly methods: RPCMethodDescriptor[];
}
export declare class GrpcServiceDescriptor {
    private readonly fileDescriptor;
    private readonly exportMap;
    private readonly pathToRoot;
    constructor(fileDescriptor: FileDescriptorProto, exportMap: ExportMap);
    readonly filename: string;
    readonly packageName: string;
    readonly imports: ImportDescriptor[];
    readonly services: RPCDescriptor[];
}
