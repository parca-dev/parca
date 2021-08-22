// package: parca.scrape
// file: scrape/scrape.proto

import * as jspb from "google-protobuf";
import * as google_api_annotations_pb from "../google/api/annotations_pb";

export class TargetsRequest extends jspb.Message {
  getState(): TargetsRequest.StateMap[keyof TargetsRequest.StateMap];
  setState(value: TargetsRequest.StateMap[keyof TargetsRequest.StateMap]): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): TargetsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: TargetsRequest): TargetsRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: TargetsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): TargetsRequest;
  static deserializeBinaryFromReader(message: TargetsRequest, reader: jspb.BinaryReader): TargetsRequest;
}

export namespace TargetsRequest {
  export type AsObject = {
    state: TargetsRequest.StateMap[keyof TargetsRequest.StateMap],
  }

  export interface StateMap {
    ANY: 0;
    ACTIVE: 1;
    DROPPED: 2;
  }

  export const State: StateMap;
}

export class TargetsResponse extends jspb.Message {
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): TargetsResponse.AsObject;
  static toObject(includeInstance: boolean, msg: TargetsResponse): TargetsResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: TargetsResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): TargetsResponse;
  static deserializeBinaryFromReader(message: TargetsResponse, reader: jspb.BinaryReader): TargetsResponse;
}

export namespace TargetsResponse {
  export type AsObject = {
  }
}

