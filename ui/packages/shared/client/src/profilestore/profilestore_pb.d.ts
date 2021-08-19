// package: parca.profilestore
// file: profilestore/profilestore.proto

import * as jspb from "google-protobuf";
import * as google_api_annotations_pb from "../google/api/annotations_pb";
import * as google_protobuf_timestamp_pb from "google-protobuf/google/protobuf/timestamp_pb";

export class WriteRawRequest extends jspb.Message {
  getTenant(): string;
  setTenant(value: string): void;

  clearSeriesList(): void;
  getSeriesList(): Array<RawProfileSeries>;
  setSeriesList(value: Array<RawProfileSeries>): void;
  addSeries(value?: RawProfileSeries, index?: number): RawProfileSeries;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): WriteRawRequest.AsObject;
  static toObject(includeInstance: boolean, msg: WriteRawRequest): WriteRawRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: WriteRawRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): WriteRawRequest;
  static deserializeBinaryFromReader(message: WriteRawRequest, reader: jspb.BinaryReader): WriteRawRequest;
}

export namespace WriteRawRequest {
  export type AsObject = {
    tenant: string,
    seriesList: Array<RawProfileSeries.AsObject>,
  }
}

export class WriteRawResponse extends jspb.Message {
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): WriteRawResponse.AsObject;
  static toObject(includeInstance: boolean, msg: WriteRawResponse): WriteRawResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: WriteRawResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): WriteRawResponse;
  static deserializeBinaryFromReader(message: WriteRawResponse, reader: jspb.BinaryReader): WriteRawResponse;
}

export namespace WriteRawResponse {
  export type AsObject = {
  }
}

export class RawProfileSeries extends jspb.Message {
  hasLabels(): boolean;
  clearLabels(): void;
  getLabels(): LabelSet | undefined;
  setLabels(value?: LabelSet): void;

  clearSamplesList(): void;
  getSamplesList(): Array<RawSample>;
  setSamplesList(value: Array<RawSample>): void;
  addSamples(value?: RawSample, index?: number): RawSample;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): RawProfileSeries.AsObject;
  static toObject(includeInstance: boolean, msg: RawProfileSeries): RawProfileSeries.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: RawProfileSeries, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): RawProfileSeries;
  static deserializeBinaryFromReader(message: RawProfileSeries, reader: jspb.BinaryReader): RawProfileSeries;
}

export namespace RawProfileSeries {
  export type AsObject = {
    labels?: LabelSet.AsObject,
    samplesList: Array<RawSample.AsObject>,
  }
}

export class Label extends jspb.Message {
  getName(): string;
  setName(value: string): void;

  getValue(): string;
  setValue(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Label.AsObject;
  static toObject(includeInstance: boolean, msg: Label): Label.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: Label, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Label;
  static deserializeBinaryFromReader(message: Label, reader: jspb.BinaryReader): Label;
}

export namespace Label {
  export type AsObject = {
    name: string,
    value: string,
  }
}

export class LabelSet extends jspb.Message {
  clearLabelsList(): void;
  getLabelsList(): Array<Label>;
  setLabelsList(value: Array<Label>): void;
  addLabels(value?: Label, index?: number): Label;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): LabelSet.AsObject;
  static toObject(includeInstance: boolean, msg: LabelSet): LabelSet.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: LabelSet, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): LabelSet;
  static deserializeBinaryFromReader(message: LabelSet, reader: jspb.BinaryReader): LabelSet;
}

export namespace LabelSet {
  export type AsObject = {
    labelsList: Array<Label.AsObject>,
  }
}

export class RawSample extends jspb.Message {
  getRawProfile(): Uint8Array | string;
  getRawProfile_asU8(): Uint8Array;
  getRawProfile_asB64(): string;
  setRawProfile(value: Uint8Array | string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): RawSample.AsObject;
  static toObject(includeInstance: boolean, msg: RawSample): RawSample.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: RawSample, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): RawSample;
  static deserializeBinaryFromReader(message: RawSample, reader: jspb.BinaryReader): RawSample;
}

export namespace RawSample {
  export type AsObject = {
    rawProfile: Uint8Array | string,
  }
}

