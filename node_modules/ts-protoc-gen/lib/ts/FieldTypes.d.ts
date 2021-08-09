import { ExportMap } from "../ExportMap";
import { FieldDescriptorProto } from "google-protobuf/google/protobuf/descriptor_pb";
export declare const MESSAGE_TYPE = 11;
export declare const BYTES_TYPE = 12;
export declare const ENUM_TYPE = 14;
export declare function getTypeName(fieldTypeNum: number): string;
export declare function getFieldType(type: FieldDescriptorProto.Type, typeName: string | null, currentFileName: string, exportMap: ExportMap): string;
