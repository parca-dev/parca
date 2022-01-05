// package: parca.metastore.v1alpha1
// file: parca/metastore/v1alpha1/metastore.proto

import * as jspb from 'google-protobuf';

export class Sample extends jspb.Message {
  clearLocationIdsList(): void;
  getLocationIdsList(): Array<Uint8Array | string>;
  getLocationIdsList_asU8(): Array<Uint8Array>;
  getLocationIdsList_asB64(): Array<string>;
  setLocationIdsList(value: Array<Uint8Array | string>): void;
  addLocationIds(value: Uint8Array | string, index?: number): Uint8Array | string;

  getLabelsMap(): jspb.Map<string, SampleLabel>;
  clearLabelsMap(): void;
  getNumLabelsMap(): jspb.Map<string, SampleNumLabel>;
  clearNumLabelsMap(): void;
  getNumUnitsMap(): jspb.Map<string, SampleNumUnit>;
  clearNumUnitsMap(): void;
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Sample.AsObject;
  static toObject(includeInstance: boolean, msg: Sample): Sample.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: Sample, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Sample;
  static deserializeBinaryFromReader(message: Sample, reader: jspb.BinaryReader): Sample;
}

export namespace Sample {
  export type AsObject = {
    locationIdsList: Array<Uint8Array | string>;
    labelsMap: Array<[string, SampleLabel.AsObject]>;
    numLabelsMap: Array<[string, SampleNumLabel.AsObject]>;
    numUnitsMap: Array<[string, SampleNumUnit.AsObject]>;
  };
}

export class SampleLabel extends jspb.Message {
  clearLabelsList(): void;
  getLabelsList(): Array<string>;
  setLabelsList(value: Array<string>): void;
  addLabels(value: string, index?: number): string;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): SampleLabel.AsObject;
  static toObject(includeInstance: boolean, msg: SampleLabel): SampleLabel.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: SampleLabel, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): SampleLabel;
  static deserializeBinaryFromReader(message: SampleLabel, reader: jspb.BinaryReader): SampleLabel;
}

export namespace SampleLabel {
  export type AsObject = {
    labelsList: Array<string>;
  };
}

export class SampleNumLabel extends jspb.Message {
  clearNumLabelsList(): void;
  getNumLabelsList(): Array<number>;
  setNumLabelsList(value: Array<number>): void;
  addNumLabels(value: number, index?: number): number;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): SampleNumLabel.AsObject;
  static toObject(includeInstance: boolean, msg: SampleNumLabel): SampleNumLabel.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: SampleNumLabel, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): SampleNumLabel;
  static deserializeBinaryFromReader(
    message: SampleNumLabel,
    reader: jspb.BinaryReader
  ): SampleNumLabel;
}

export namespace SampleNumLabel {
  export type AsObject = {
    numLabelsList: Array<number>;
  };
}

export class SampleNumUnit extends jspb.Message {
  clearUnitsList(): void;
  getUnitsList(): Array<string>;
  setUnitsList(value: Array<string>): void;
  addUnits(value: string, index?: number): string;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): SampleNumUnit.AsObject;
  static toObject(includeInstance: boolean, msg: SampleNumUnit): SampleNumUnit.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: SampleNumUnit, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): SampleNumUnit;
  static deserializeBinaryFromReader(
    message: SampleNumUnit,
    reader: jspb.BinaryReader
  ): SampleNumUnit;
}

export namespace SampleNumUnit {
  export type AsObject = {
    unitsList: Array<string>;
  };
}

export class Location extends jspb.Message {
  getId(): Uint8Array | string;
  getId_asU8(): Uint8Array;
  getId_asB64(): string;
  setId(value: Uint8Array | string): void;

  getAddress(): number;
  setAddress(value: number): void;

  getMappingId(): Uint8Array | string;
  getMappingId_asU8(): Uint8Array;
  getMappingId_asB64(): string;
  setMappingId(value: Uint8Array | string): void;

  getIsFolded(): boolean;
  setIsFolded(value: boolean): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Location.AsObject;
  static toObject(includeInstance: boolean, msg: Location): Location.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: Location, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Location;
  static deserializeBinaryFromReader(message: Location, reader: jspb.BinaryReader): Location;
}

export namespace Location {
  export type AsObject = {
    id: Uint8Array | string;
    address: number;
    mappingId: Uint8Array | string;
    isFolded: boolean;
  };
}

export class LocationLines extends jspb.Message {
  getId(): Uint8Array | string;
  getId_asU8(): Uint8Array;
  getId_asB64(): string;
  setId(value: Uint8Array | string): void;

  clearLinesList(): void;
  getLinesList(): Array<Line>;
  setLinesList(value: Array<Line>): void;
  addLines(value?: Line, index?: number): Line;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): LocationLines.AsObject;
  static toObject(includeInstance: boolean, msg: LocationLines): LocationLines.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: LocationLines, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): LocationLines;
  static deserializeBinaryFromReader(
    message: LocationLines,
    reader: jspb.BinaryReader
  ): LocationLines;
}

export namespace LocationLines {
  export type AsObject = {
    id: Uint8Array | string;
    linesList: Array<Line.AsObject>;
  };
}

export class Line extends jspb.Message {
  getFunctionId(): Uint8Array | string;
  getFunctionId_asU8(): Uint8Array;
  getFunctionId_asB64(): string;
  setFunctionId(value: Uint8Array | string): void;

  getLine(): number;
  setLine(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Line.AsObject;
  static toObject(includeInstance: boolean, msg: Line): Line.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: Line, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Line;
  static deserializeBinaryFromReader(message: Line, reader: jspb.BinaryReader): Line;
}

export namespace Line {
  export type AsObject = {
    functionId: Uint8Array | string;
    line: number;
  };
}

export class Function extends jspb.Message {
  getId(): Uint8Array | string;
  getId_asU8(): Uint8Array;
  getId_asB64(): string;
  setId(value: Uint8Array | string): void;

  getStartLine(): number;
  setStartLine(value: number): void;

  getName(): string;
  setName(value: string): void;

  getSystemName(): string;
  setSystemName(value: string): void;

  getFilename(): string;
  setFilename(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Function.AsObject;
  static toObject(includeInstance: boolean, msg: Function): Function.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: Function, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Function;
  static deserializeBinaryFromReader(message: Function, reader: jspb.BinaryReader): Function;
}

export namespace Function {
  export type AsObject = {
    id: Uint8Array | string;
    startLine: number;
    name: string;
    systemName: string;
    filename: string;
  };
}

export class Mapping extends jspb.Message {
  getId(): Uint8Array | string;
  getId_asU8(): Uint8Array;
  getId_asB64(): string;
  setId(value: Uint8Array | string): void;

  getStart(): number;
  setStart(value: number): void;

  getLimit(): number;
  setLimit(value: number): void;

  getOffset(): number;
  setOffset(value: number): void;

  getFile(): string;
  setFile(value: string): void;

  getBuildId(): string;
  setBuildId(value: string): void;

  getHasFunctions(): boolean;
  setHasFunctions(value: boolean): void;

  getHasFilenames(): boolean;
  setHasFilenames(value: boolean): void;

  getHasLineNumbers(): boolean;
  setHasLineNumbers(value: boolean): void;

  getHasInlineFrames(): boolean;
  setHasInlineFrames(value: boolean): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Mapping.AsObject;
  static toObject(includeInstance: boolean, msg: Mapping): Mapping.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: Mapping, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Mapping;
  static deserializeBinaryFromReader(message: Mapping, reader: jspb.BinaryReader): Mapping;
}

export namespace Mapping {
  export type AsObject = {
    id: Uint8Array | string;
    start: number;
    limit: number;
    offset: number;
    file: string;
    buildId: string;
    hasFunctions: boolean;
    hasFilenames: boolean;
    hasLineNumbers: boolean;
    hasInlineFrames: boolean;
  };
}
