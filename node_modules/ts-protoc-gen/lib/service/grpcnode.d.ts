import { ExportMap } from "../ExportMap";
import { FileDescriptorProto } from "google-protobuf/google/protobuf/descriptor_pb";
import { CodeGeneratorResponse } from "google-protobuf/google/protobuf/compiler/plugin_pb";
import { ModeParameter } from "../parameters";
export declare function generateGrpcNodeService(filename: string, descriptor: FileDescriptorProto, exportMap: ExportMap, modeParameter: ModeParameter): CodeGeneratorResponse.File;
