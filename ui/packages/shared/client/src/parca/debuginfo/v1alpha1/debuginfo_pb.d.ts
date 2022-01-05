// package: parca.debuginfo.v1alpha1
// file: parca/debuginfo/v1alpha1/debuginfo.proto

import * as jspb from 'google-protobuf';

export class ExistsRequest extends jspb.Message {
  getBuildId(): string;
  setBuildId(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ExistsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: ExistsRequest): ExistsRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ExistsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ExistsRequest;
  static deserializeBinaryFromReader(
    message: ExistsRequest,
    reader: jspb.BinaryReader
  ): ExistsRequest;
}

export namespace ExistsRequest {
  export type AsObject = {
    buildId: string;
  };
}

export class ExistsResponse extends jspb.Message {
  getExists(): boolean;
  setExists(value: boolean): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ExistsResponse.AsObject;
  static toObject(includeInstance: boolean, msg: ExistsResponse): ExistsResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ExistsResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ExistsResponse;
  static deserializeBinaryFromReader(
    message: ExistsResponse,
    reader: jspb.BinaryReader
  ): ExistsResponse;
}

export namespace ExistsResponse {
  export type AsObject = {
    exists: boolean;
  };
}

export class UploadRequest extends jspb.Message {
  hasInfo(): boolean;
  clearInfo(): void;
  getInfo(): UploadInfo | undefined;
  setInfo(value?: UploadInfo): void;

  hasChunkData(): boolean;
  clearChunkData(): void;
  getChunkData(): Uint8Array | string;
  getChunkData_asU8(): Uint8Array;
  getChunkData_asB64(): string;
  setChunkData(value: Uint8Array | string): void;

  getDataCase(): UploadRequest.DataCase;
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): UploadRequest.AsObject;
  static toObject(includeInstance: boolean, msg: UploadRequest): UploadRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: UploadRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): UploadRequest;
  static deserializeBinaryFromReader(
    message: UploadRequest,
    reader: jspb.BinaryReader
  ): UploadRequest;
}

export namespace UploadRequest {
  export type AsObject = {
    info?: UploadInfo.AsObject;
    chunkData: Uint8Array | string;
  };

  export enum DataCase {
    DATA_NOT_SET = 0,
    INFO = 1,
    CHUNK_DATA = 2,
  }
}

export class UploadInfo extends jspb.Message {
  getBuildId(): string;
  setBuildId(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): UploadInfo.AsObject;
  static toObject(includeInstance: boolean, msg: UploadInfo): UploadInfo.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: UploadInfo, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): UploadInfo;
  static deserializeBinaryFromReader(message: UploadInfo, reader: jspb.BinaryReader): UploadInfo;
}

export namespace UploadInfo {
  export type AsObject = {
    buildId: string;
  };
}

export class UploadResponse extends jspb.Message {
  getBuildId(): string;
  setBuildId(value: string): void;

  getSize(): number;
  setSize(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): UploadResponse.AsObject;
  static toObject(includeInstance: boolean, msg: UploadResponse): UploadResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: UploadResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): UploadResponse;
  static deserializeBinaryFromReader(
    message: UploadResponse,
    reader: jspb.BinaryReader
  ): UploadResponse;
}

export namespace UploadResponse {
  export type AsObject = {
    buildId: string;
    size: number;
  };
}
