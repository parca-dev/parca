// package: parca.debuginfo
// file: debuginfo/debuginfo.proto

import * as jspb from "google-protobuf";

export class DebugInfoExistsRequest extends jspb.Message {
  getBuildId(): string;
  setBuildId(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DebugInfoExistsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: DebugInfoExistsRequest): DebugInfoExistsRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: DebugInfoExistsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DebugInfoExistsRequest;
  static deserializeBinaryFromReader(message: DebugInfoExistsRequest, reader: jspb.BinaryReader): DebugInfoExistsRequest;
}

export namespace DebugInfoExistsRequest {
  export type AsObject = {
    buildId: string,
  }
}

export class DebugInfoExistsResponse extends jspb.Message {
  getExists(): boolean;
  setExists(value: boolean): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DebugInfoExistsResponse.AsObject;
  static toObject(includeInstance: boolean, msg: DebugInfoExistsResponse): DebugInfoExistsResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: DebugInfoExistsResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DebugInfoExistsResponse;
  static deserializeBinaryFromReader(message: DebugInfoExistsResponse, reader: jspb.BinaryReader): DebugInfoExistsResponse;
}

export namespace DebugInfoExistsResponse {
  export type AsObject = {
    exists: boolean,
  }
}

export class DebugInfoUploadRequest extends jspb.Message {
  hasInfo(): boolean;
  clearInfo(): void;
  getInfo(): DebugInfoUploadInfo | undefined;
  setInfo(value?: DebugInfoUploadInfo): void;

  hasChunkData(): boolean;
  clearChunkData(): void;
  getChunkData(): Uint8Array | string;
  getChunkData_asU8(): Uint8Array;
  getChunkData_asB64(): string;
  setChunkData(value: Uint8Array | string): void;

  getDataCase(): DebugInfoUploadRequest.DataCase;
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DebugInfoUploadRequest.AsObject;
  static toObject(includeInstance: boolean, msg: DebugInfoUploadRequest): DebugInfoUploadRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: DebugInfoUploadRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DebugInfoUploadRequest;
  static deserializeBinaryFromReader(message: DebugInfoUploadRequest, reader: jspb.BinaryReader): DebugInfoUploadRequest;
}

export namespace DebugInfoUploadRequest {
  export type AsObject = {
    info?: DebugInfoUploadInfo.AsObject,
    chunkData: Uint8Array | string,
  }

  export enum DataCase {
    DATA_NOT_SET = 0,
    INFO = 1,
    CHUNK_DATA = 2,
  }
}

export class DebugInfoUploadInfo extends jspb.Message {
  getBuildId(): string;
  setBuildId(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DebugInfoUploadInfo.AsObject;
  static toObject(includeInstance: boolean, msg: DebugInfoUploadInfo): DebugInfoUploadInfo.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: DebugInfoUploadInfo, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DebugInfoUploadInfo;
  static deserializeBinaryFromReader(message: DebugInfoUploadInfo, reader: jspb.BinaryReader): DebugInfoUploadInfo;
}

export namespace DebugInfoUploadInfo {
  export type AsObject = {
    buildId: string,
  }
}

export class DebugInfoUploadResponse extends jspb.Message {
  getBuildId(): string;
  setBuildId(value: string): void;

  getSize(): number;
  setSize(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DebugInfoUploadResponse.AsObject;
  static toObject(includeInstance: boolean, msg: DebugInfoUploadResponse): DebugInfoUploadResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: DebugInfoUploadResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DebugInfoUploadResponse;
  static deserializeBinaryFromReader(message: DebugInfoUploadResponse, reader: jspb.BinaryReader): DebugInfoUploadResponse;
}

export namespace DebugInfoUploadResponse {
  export type AsObject = {
    buildId: string,
    size: number,
  }
}

