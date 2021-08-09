import { FileDescriptorProto, DescriptorProto, MessageOptions, EnumOptions, FieldDescriptorProto } from "google-protobuf/google/protobuf/descriptor_pb";
import Type = FieldDescriptorProto.Type;
declare type MapFieldOptions = {
    key: [Type, string | null];
    value: [Type, string | null];
};
export declare type ExportMessageEntry = {
    pkg: string;
    fileName: string;
    messageOptions?: MessageOptions;
    mapFieldOptions?: MapFieldOptions;
};
export declare type ExportEnumEntry = {
    pkg: string;
    fileName: string;
    enumOptions?: EnumOptions;
};
export declare class ExportMap {
    messageMap: {
        [key: string]: ExportMessageEntry;
    };
    enumMap: {
        [key: string]: ExportEnumEntry;
    };
    exportNested(scope: string, fileDescriptor: FileDescriptorProto, message: DescriptorProto): void;
    addFileDescriptor(fileDescriptor: FileDescriptorProto): void;
    getMessage(str: string): ExportMessageEntry | undefined;
    getEnum(str: string): ExportEnumEntry | undefined;
}
export {};
