// package: perftools.profiles
// file: google/pprof/profile.proto

import * as jspb from "google-protobuf";

export class Profile extends jspb.Message {
  clearSampleTypeList(): void;
  getSampleTypeList(): Array<ValueType>;
  setSampleTypeList(value: Array<ValueType>): void;
  addSampleType(value?: ValueType, index?: number): ValueType;

  clearSampleList(): void;
  getSampleList(): Array<Sample>;
  setSampleList(value: Array<Sample>): void;
  addSample(value?: Sample, index?: number): Sample;

  clearMappingList(): void;
  getMappingList(): Array<Mapping>;
  setMappingList(value: Array<Mapping>): void;
  addMapping(value?: Mapping, index?: number): Mapping;

  clearLocationList(): void;
  getLocationList(): Array<Location>;
  setLocationList(value: Array<Location>): void;
  addLocation(value?: Location, index?: number): Location;

  clearFunctionList(): void;
  getFunctionList(): Array<Function>;
  setFunctionList(value: Array<Function>): void;
  addFunction(value?: Function, index?: number): Function;

  clearStringTableList(): void;
  getStringTableList(): Array<string>;
  setStringTableList(value: Array<string>): void;
  addStringTable(value: string, index?: number): string;

  getDropFrames(): number;
  setDropFrames(value: number): void;

  getKeepFrames(): number;
  setKeepFrames(value: number): void;

  getTimeNanos(): number;
  setTimeNanos(value: number): void;

  getDurationNanos(): number;
  setDurationNanos(value: number): void;

  hasPeriodType(): boolean;
  clearPeriodType(): void;
  getPeriodType(): ValueType | undefined;
  setPeriodType(value?: ValueType): void;

  getPeriod(): number;
  setPeriod(value: number): void;

  clearCommentList(): void;
  getCommentList(): Array<number>;
  setCommentList(value: Array<number>): void;
  addComment(value: number, index?: number): number;

  getDefaultSampleType(): number;
  setDefaultSampleType(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Profile.AsObject;
  static toObject(includeInstance: boolean, msg: Profile): Profile.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: Profile, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Profile;
  static deserializeBinaryFromReader(message: Profile, reader: jspb.BinaryReader): Profile;
}

export namespace Profile {
  export type AsObject = {
    sampleTypeList: Array<ValueType.AsObject>,
    sampleList: Array<Sample.AsObject>,
    mappingList: Array<Mapping.AsObject>,
    locationList: Array<Location.AsObject>,
    functionList: Array<Function.AsObject>,
    stringTableList: Array<string>,
    dropFrames: number,
    keepFrames: number,
    timeNanos: number,
    durationNanos: number,
    periodType?: ValueType.AsObject,
    period: number,
    commentList: Array<number>,
    defaultSampleType: number,
  }
}

export class ValueType extends jspb.Message {
  getType(): number;
  setType(value: number): void;

  getUnit(): number;
  setUnit(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ValueType.AsObject;
  static toObject(includeInstance: boolean, msg: ValueType): ValueType.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ValueType, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ValueType;
  static deserializeBinaryFromReader(message: ValueType, reader: jspb.BinaryReader): ValueType;
}

export namespace ValueType {
  export type AsObject = {
    type: number,
    unit: number,
  }
}

export class Sample extends jspb.Message {
  clearLocationIdList(): void;
  getLocationIdList(): Array<number>;
  setLocationIdList(value: Array<number>): void;
  addLocationId(value: number, index?: number): number;

  clearValueList(): void;
  getValueList(): Array<number>;
  setValueList(value: Array<number>): void;
  addValue(value: number, index?: number): number;

  clearLabelList(): void;
  getLabelList(): Array<Label>;
  setLabelList(value: Array<Label>): void;
  addLabel(value?: Label, index?: number): Label;

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
    locationIdList: Array<number>,
    valueList: Array<number>,
    labelList: Array<Label.AsObject>,
  }
}

export class Label extends jspb.Message {
  getKey(): number;
  setKey(value: number): void;

  getStr(): number;
  setStr(value: number): void;

  getNum(): number;
  setNum(value: number): void;

  getNumUnit(): number;
  setNumUnit(value: number): void;

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
    key: number,
    str: number,
    num: number,
    numUnit: number,
  }
}

export class Mapping extends jspb.Message {
  getId(): number;
  setId(value: number): void;

  getMemoryStart(): number;
  setMemoryStart(value: number): void;

  getMemoryLimit(): number;
  setMemoryLimit(value: number): void;

  getFileOffset(): number;
  setFileOffset(value: number): void;

  getFilename(): number;
  setFilename(value: number): void;

  getBuildId(): number;
  setBuildId(value: number): void;

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
    id: number,
    memoryStart: number,
    memoryLimit: number,
    fileOffset: number,
    filename: number,
    buildId: number,
    hasFunctions: boolean,
    hasFilenames: boolean,
    hasLineNumbers: boolean,
    hasInlineFrames: boolean,
  }
}

export class Location extends jspb.Message {
  getId(): number;
  setId(value: number): void;

  getMappingId(): number;
  setMappingId(value: number): void;

  getAddress(): number;
  setAddress(value: number): void;

  clearLineList(): void;
  getLineList(): Array<Line>;
  setLineList(value: Array<Line>): void;
  addLine(value?: Line, index?: number): Line;

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
    id: number,
    mappingId: number,
    address: number,
    lineList: Array<Line.AsObject>,
    isFolded: boolean,
  }
}

export class Line extends jspb.Message {
  getFunctionId(): number;
  setFunctionId(value: number): void;

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
    functionId: number,
    line: number,
  }
}

export class Function extends jspb.Message {
  getId(): number;
  setId(value: number): void;

  getName(): number;
  setName(value: number): void;

  getSystemName(): number;
  setSystemName(value: number): void;

  getFilename(): number;
  setFilename(value: number): void;

  getStartLine(): number;
  setStartLine(value: number): void;

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
    id: number,
    name: number,
    systemName: number,
    filename: number,
    startLine: number,
  }
}

