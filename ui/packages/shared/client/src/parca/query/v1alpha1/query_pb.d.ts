// package: parca.query.v1alpha1
// file: parca/query/v1alpha1/query.proto

import * as jspb from "google-protobuf";
import * as google_api_annotations_pb from "../../../google/api/annotations_pb";
import * as google_protobuf_timestamp_pb from "google-protobuf/google/protobuf/timestamp_pb";
import * as parca_profilestore_v1alpha1_profilestore_pb from "../../../parca/profilestore/v1alpha1/profilestore_pb";
import * as parca_metastore_v1alpha1_metastore_pb from "../../../parca/metastore/v1alpha1/metastore_pb";

export class QueryRangeRequest extends jspb.Message {
  getQuery(): string;
  setQuery(value: string): void;

  hasStart(): boolean;
  clearStart(): void;
  getStart(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setStart(value?: google_protobuf_timestamp_pb.Timestamp): void;

  hasEnd(): boolean;
  clearEnd(): void;
  getEnd(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setEnd(value?: google_protobuf_timestamp_pb.Timestamp): void;

  getLimit(): number;
  setLimit(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): QueryRangeRequest.AsObject;
  static toObject(includeInstance: boolean, msg: QueryRangeRequest): QueryRangeRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: QueryRangeRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): QueryRangeRequest;
  static deserializeBinaryFromReader(message: QueryRangeRequest, reader: jspb.BinaryReader): QueryRangeRequest;
}

export namespace QueryRangeRequest {
  export type AsObject = {
    query: string,
    start?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    end?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    limit: number,
  }
}

export class QueryRangeResponse extends jspb.Message {
  clearSeriesList(): void;
  getSeriesList(): Array<MetricsSeries>;
  setSeriesList(value: Array<MetricsSeries>): void;
  addSeries(value?: MetricsSeries, index?: number): MetricsSeries;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): QueryRangeResponse.AsObject;
  static toObject(includeInstance: boolean, msg: QueryRangeResponse): QueryRangeResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: QueryRangeResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): QueryRangeResponse;
  static deserializeBinaryFromReader(message: QueryRangeResponse, reader: jspb.BinaryReader): QueryRangeResponse;
}

export namespace QueryRangeResponse {
  export type AsObject = {
    seriesList: Array<MetricsSeries.AsObject>,
  }
}

export class MetricsSeries extends jspb.Message {
  hasLabelset(): boolean;
  clearLabelset(): void;
  getLabelset(): parca_profilestore_v1alpha1_profilestore_pb.LabelSet | undefined;
  setLabelset(value?: parca_profilestore_v1alpha1_profilestore_pb.LabelSet): void;

  clearSamplesList(): void;
  getSamplesList(): Array<MetricsSample>;
  setSamplesList(value: Array<MetricsSample>): void;
  addSamples(value?: MetricsSample, index?: number): MetricsSample;

  hasPeriodType(): boolean;
  clearPeriodType(): void;
  getPeriodType(): ValueType | undefined;
  setPeriodType(value?: ValueType): void;

  hasSampleType(): boolean;
  clearSampleType(): void;
  getSampleType(): ValueType | undefined;
  setSampleType(value?: ValueType): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): MetricsSeries.AsObject;
  static toObject(includeInstance: boolean, msg: MetricsSeries): MetricsSeries.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: MetricsSeries, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): MetricsSeries;
  static deserializeBinaryFromReader(message: MetricsSeries, reader: jspb.BinaryReader): MetricsSeries;
}

export namespace MetricsSeries {
  export type AsObject = {
    labelset?: parca_profilestore_v1alpha1_profilestore_pb.LabelSet.AsObject,
    samplesList: Array<MetricsSample.AsObject>,
    periodType?: ValueType.AsObject,
    sampleType?: ValueType.AsObject,
  }
}

export class MetricsSample extends jspb.Message {
  hasTimestamp(): boolean;
  clearTimestamp(): void;
  getTimestamp(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setTimestamp(value?: google_protobuf_timestamp_pb.Timestamp): void;

  getValue(): number;
  setValue(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): MetricsSample.AsObject;
  static toObject(includeInstance: boolean, msg: MetricsSample): MetricsSample.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: MetricsSample, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): MetricsSample;
  static deserializeBinaryFromReader(message: MetricsSample, reader: jspb.BinaryReader): MetricsSample;
}

export namespace MetricsSample {
  export type AsObject = {
    timestamp?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    value: number,
  }
}

export class MergeProfile extends jspb.Message {
  getQuery(): string;
  setQuery(value: string): void;

  hasStart(): boolean;
  clearStart(): void;
  getStart(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setStart(value?: google_protobuf_timestamp_pb.Timestamp): void;

  hasEnd(): boolean;
  clearEnd(): void;
  getEnd(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setEnd(value?: google_protobuf_timestamp_pb.Timestamp): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): MergeProfile.AsObject;
  static toObject(includeInstance: boolean, msg: MergeProfile): MergeProfile.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: MergeProfile, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): MergeProfile;
  static deserializeBinaryFromReader(message: MergeProfile, reader: jspb.BinaryReader): MergeProfile;
}

export namespace MergeProfile {
  export type AsObject = {
    query: string,
    start?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    end?: google_protobuf_timestamp_pb.Timestamp.AsObject,
  }
}

export class SingleProfile extends jspb.Message {
  hasTime(): boolean;
  clearTime(): void;
  getTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setTime(value?: google_protobuf_timestamp_pb.Timestamp): void;

  getQuery(): string;
  setQuery(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): SingleProfile.AsObject;
  static toObject(includeInstance: boolean, msg: SingleProfile): SingleProfile.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: SingleProfile, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): SingleProfile;
  static deserializeBinaryFromReader(message: SingleProfile, reader: jspb.BinaryReader): SingleProfile;
}

export namespace SingleProfile {
  export type AsObject = {
    time?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    query: string,
  }
}

export class DiffProfile extends jspb.Message {
  hasA(): boolean;
  clearA(): void;
  getA(): ProfileDiffSelection | undefined;
  setA(value?: ProfileDiffSelection): void;

  hasB(): boolean;
  clearB(): void;
  getB(): ProfileDiffSelection | undefined;
  setB(value?: ProfileDiffSelection): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DiffProfile.AsObject;
  static toObject(includeInstance: boolean, msg: DiffProfile): DiffProfile.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: DiffProfile, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DiffProfile;
  static deserializeBinaryFromReader(message: DiffProfile, reader: jspb.BinaryReader): DiffProfile;
}

export namespace DiffProfile {
  export type AsObject = {
    a?: ProfileDiffSelection.AsObject,
    b?: ProfileDiffSelection.AsObject,
  }
}

export class ProfileDiffSelection extends jspb.Message {
  getMode(): ProfileDiffSelection.ModeMap[keyof ProfileDiffSelection.ModeMap];
  setMode(value: ProfileDiffSelection.ModeMap[keyof ProfileDiffSelection.ModeMap]): void;

  hasMerge(): boolean;
  clearMerge(): void;
  getMerge(): MergeProfile | undefined;
  setMerge(value?: MergeProfile): void;

  hasSingle(): boolean;
  clearSingle(): void;
  getSingle(): SingleProfile | undefined;
  setSingle(value?: SingleProfile): void;

  getOptionsCase(): ProfileDiffSelection.OptionsCase;
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ProfileDiffSelection.AsObject;
  static toObject(includeInstance: boolean, msg: ProfileDiffSelection): ProfileDiffSelection.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ProfileDiffSelection, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ProfileDiffSelection;
  static deserializeBinaryFromReader(message: ProfileDiffSelection, reader: jspb.BinaryReader): ProfileDiffSelection;
}

export namespace ProfileDiffSelection {
  export type AsObject = {
    mode: ProfileDiffSelection.ModeMap[keyof ProfileDiffSelection.ModeMap],
    merge?: MergeProfile.AsObject,
    single?: SingleProfile.AsObject,
  }

  export interface ModeMap {
    MODE_SINGLE_UNSPECIFIED: 0;
    MODE_MERGE: 1;
  }

  export const Mode: ModeMap;

  export enum OptionsCase {
    OPTIONS_NOT_SET = 0,
    MERGE = 2,
    SINGLE = 3,
  }
}

export class QueryRequest extends jspb.Message {
  getMode(): QueryRequest.ModeMap[keyof QueryRequest.ModeMap];
  setMode(value: QueryRequest.ModeMap[keyof QueryRequest.ModeMap]): void;

  hasDiff(): boolean;
  clearDiff(): void;
  getDiff(): DiffProfile | undefined;
  setDiff(value?: DiffProfile): void;

  hasMerge(): boolean;
  clearMerge(): void;
  getMerge(): MergeProfile | undefined;
  setMerge(value?: MergeProfile): void;

  hasSingle(): boolean;
  clearSingle(): void;
  getSingle(): SingleProfile | undefined;
  setSingle(value?: SingleProfile): void;

  getReportType(): QueryRequest.ReportTypeMap[keyof QueryRequest.ReportTypeMap];
  setReportType(value: QueryRequest.ReportTypeMap[keyof QueryRequest.ReportTypeMap]): void;

  getOptionsCase(): QueryRequest.OptionsCase;
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): QueryRequest.AsObject;
  static toObject(includeInstance: boolean, msg: QueryRequest): QueryRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: QueryRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): QueryRequest;
  static deserializeBinaryFromReader(message: QueryRequest, reader: jspb.BinaryReader): QueryRequest;
}

export namespace QueryRequest {
  export type AsObject = {
    mode: QueryRequest.ModeMap[keyof QueryRequest.ModeMap],
    diff?: DiffProfile.AsObject,
    merge?: MergeProfile.AsObject,
    single?: SingleProfile.AsObject,
    reportType: QueryRequest.ReportTypeMap[keyof QueryRequest.ReportTypeMap],
  }

  export interface ModeMap {
    MODE_SINGLE_UNSPECIFIED: 0;
    MODE_DIFF: 1;
    MODE_MERGE: 2;
  }

  export const Mode: ModeMap;

  export interface ReportTypeMap {
    REPORT_TYPE_FLAMEGRAPH_UNSPECIFIED: 0;
    REPORT_TYPE_PPROF_UNSPECIFIED: 1;
  }

  export const ReportType: ReportTypeMap;

  export enum OptionsCase {
    OPTIONS_NOT_SET = 0,
    DIFF = 2,
    MERGE = 3,
    SINGLE = 4,
  }
}

export class Flamegraph extends jspb.Message {
  hasRoot(): boolean;
  clearRoot(): void;
  getRoot(): FlamegraphRootNode | undefined;
  setRoot(value?: FlamegraphRootNode): void;

  getTotal(): number;
  setTotal(value: number): void;

  getUnit(): string;
  setUnit(value: string): void;

  getHeight(): number;
  setHeight(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Flamegraph.AsObject;
  static toObject(includeInstance: boolean, msg: Flamegraph): Flamegraph.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: Flamegraph, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Flamegraph;
  static deserializeBinaryFromReader(message: Flamegraph, reader: jspb.BinaryReader): Flamegraph;
}

export namespace Flamegraph {
  export type AsObject = {
    root?: FlamegraphRootNode.AsObject,
    total: number,
    unit: string,
    height: number,
  }
}

export class FlamegraphRootNode extends jspb.Message {
  getCumulative(): number;
  setCumulative(value: number): void;

  getDiff(): number;
  setDiff(value: number): void;

  clearChildrenList(): void;
  getChildrenList(): Array<FlamegraphNode>;
  setChildrenList(value: Array<FlamegraphNode>): void;
  addChildren(value?: FlamegraphNode, index?: number): FlamegraphNode;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): FlamegraphRootNode.AsObject;
  static toObject(includeInstance: boolean, msg: FlamegraphRootNode): FlamegraphRootNode.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: FlamegraphRootNode, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): FlamegraphRootNode;
  static deserializeBinaryFromReader(message: FlamegraphRootNode, reader: jspb.BinaryReader): FlamegraphRootNode;
}

export namespace FlamegraphRootNode {
  export type AsObject = {
    cumulative: number,
    diff: number,
    childrenList: Array<FlamegraphNode.AsObject>,
  }
}

export class FlamegraphNode extends jspb.Message {
  hasMeta(): boolean;
  clearMeta(): void;
  getMeta(): FlamegraphNodeMeta | undefined;
  setMeta(value?: FlamegraphNodeMeta): void;

  getCumulative(): number;
  setCumulative(value: number): void;

  getDiff(): number;
  setDiff(value: number): void;

  clearChildrenList(): void;
  getChildrenList(): Array<FlamegraphNode>;
  setChildrenList(value: Array<FlamegraphNode>): void;
  addChildren(value?: FlamegraphNode, index?: number): FlamegraphNode;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): FlamegraphNode.AsObject;
  static toObject(includeInstance: boolean, msg: FlamegraphNode): FlamegraphNode.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: FlamegraphNode, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): FlamegraphNode;
  static deserializeBinaryFromReader(message: FlamegraphNode, reader: jspb.BinaryReader): FlamegraphNode;
}

export namespace FlamegraphNode {
  export type AsObject = {
    meta?: FlamegraphNodeMeta.AsObject,
    cumulative: number,
    diff: number,
    childrenList: Array<FlamegraphNode.AsObject>,
  }
}

export class FlamegraphNodeMeta extends jspb.Message {
  hasLocation(): boolean;
  clearLocation(): void;
  getLocation(): parca_metastore_v1alpha1_metastore_pb.Location | undefined;
  setLocation(value?: parca_metastore_v1alpha1_metastore_pb.Location): void;

  hasMapping(): boolean;
  clearMapping(): void;
  getMapping(): parca_metastore_v1alpha1_metastore_pb.Mapping | undefined;
  setMapping(value?: parca_metastore_v1alpha1_metastore_pb.Mapping): void;

  hasFunction(): boolean;
  clearFunction(): void;
  getFunction(): parca_metastore_v1alpha1_metastore_pb.Function | undefined;
  setFunction(value?: parca_metastore_v1alpha1_metastore_pb.Function): void;

  hasLine(): boolean;
  clearLine(): void;
  getLine(): parca_metastore_v1alpha1_metastore_pb.Line | undefined;
  setLine(value?: parca_metastore_v1alpha1_metastore_pb.Line): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): FlamegraphNodeMeta.AsObject;
  static toObject(includeInstance: boolean, msg: FlamegraphNodeMeta): FlamegraphNodeMeta.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: FlamegraphNodeMeta, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): FlamegraphNodeMeta;
  static deserializeBinaryFromReader(message: FlamegraphNodeMeta, reader: jspb.BinaryReader): FlamegraphNodeMeta;
}

export namespace FlamegraphNodeMeta {
  export type AsObject = {
    location?: parca_metastore_v1alpha1_metastore_pb.Location.AsObject,
    mapping?: parca_metastore_v1alpha1_metastore_pb.Mapping.AsObject,
    pb_function?: parca_metastore_v1alpha1_metastore_pb.Function.AsObject,
    line?: parca_metastore_v1alpha1_metastore_pb.Line.AsObject,
  }
}

export class QueryResponse extends jspb.Message {
  hasFlamegraph(): boolean;
  clearFlamegraph(): void;
  getFlamegraph(): Flamegraph | undefined;
  setFlamegraph(value?: Flamegraph): void;

  hasPprof(): boolean;
  clearPprof(): void;
  getPprof(): Uint8Array | string;
  getPprof_asU8(): Uint8Array;
  getPprof_asB64(): string;
  setPprof(value: Uint8Array | string): void;

  getReportCase(): QueryResponse.ReportCase;
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): QueryResponse.AsObject;
  static toObject(includeInstance: boolean, msg: QueryResponse): QueryResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: QueryResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): QueryResponse;
  static deserializeBinaryFromReader(message: QueryResponse, reader: jspb.BinaryReader): QueryResponse;
}

export namespace QueryResponse {
  export type AsObject = {
    flamegraph?: Flamegraph.AsObject,
    pprof: Uint8Array | string,
  }

  export enum ReportCase {
    REPORT_NOT_SET = 0,
    FLAMEGRAPH = 5,
    PPROF = 6,
  }
}

export class SeriesRequest extends jspb.Message {
  clearMatchList(): void;
  getMatchList(): Array<string>;
  setMatchList(value: Array<string>): void;
  addMatch(value: string, index?: number): string;

  hasStart(): boolean;
  clearStart(): void;
  getStart(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setStart(value?: google_protobuf_timestamp_pb.Timestamp): void;

  hasEnd(): boolean;
  clearEnd(): void;
  getEnd(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setEnd(value?: google_protobuf_timestamp_pb.Timestamp): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): SeriesRequest.AsObject;
  static toObject(includeInstance: boolean, msg: SeriesRequest): SeriesRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: SeriesRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): SeriesRequest;
  static deserializeBinaryFromReader(message: SeriesRequest, reader: jspb.BinaryReader): SeriesRequest;
}

export namespace SeriesRequest {
  export type AsObject = {
    matchList: Array<string>,
    start?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    end?: google_protobuf_timestamp_pb.Timestamp.AsObject,
  }
}

export class SeriesResponse extends jspb.Message {
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): SeriesResponse.AsObject;
  static toObject(includeInstance: boolean, msg: SeriesResponse): SeriesResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: SeriesResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): SeriesResponse;
  static deserializeBinaryFromReader(message: SeriesResponse, reader: jspb.BinaryReader): SeriesResponse;
}

export namespace SeriesResponse {
  export type AsObject = {
  }
}

export class LabelsRequest extends jspb.Message {
  clearMatchList(): void;
  getMatchList(): Array<string>;
  setMatchList(value: Array<string>): void;
  addMatch(value: string, index?: number): string;

  hasStart(): boolean;
  clearStart(): void;
  getStart(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setStart(value?: google_protobuf_timestamp_pb.Timestamp): void;

  hasEnd(): boolean;
  clearEnd(): void;
  getEnd(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setEnd(value?: google_protobuf_timestamp_pb.Timestamp): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): LabelsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: LabelsRequest): LabelsRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: LabelsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): LabelsRequest;
  static deserializeBinaryFromReader(message: LabelsRequest, reader: jspb.BinaryReader): LabelsRequest;
}

export namespace LabelsRequest {
  export type AsObject = {
    matchList: Array<string>,
    start?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    end?: google_protobuf_timestamp_pb.Timestamp.AsObject,
  }
}

export class LabelsResponse extends jspb.Message {
  clearLabelNamesList(): void;
  getLabelNamesList(): Array<string>;
  setLabelNamesList(value: Array<string>): void;
  addLabelNames(value: string, index?: number): string;

  clearWarningsList(): void;
  getWarningsList(): Array<string>;
  setWarningsList(value: Array<string>): void;
  addWarnings(value: string, index?: number): string;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): LabelsResponse.AsObject;
  static toObject(includeInstance: boolean, msg: LabelsResponse): LabelsResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: LabelsResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): LabelsResponse;
  static deserializeBinaryFromReader(message: LabelsResponse, reader: jspb.BinaryReader): LabelsResponse;
}

export namespace LabelsResponse {
  export type AsObject = {
    labelNamesList: Array<string>,
    warningsList: Array<string>,
  }
}

export class ValuesRequest extends jspb.Message {
  getLabelName(): string;
  setLabelName(value: string): void;

  clearMatchList(): void;
  getMatchList(): Array<string>;
  setMatchList(value: Array<string>): void;
  addMatch(value: string, index?: number): string;

  hasStart(): boolean;
  clearStart(): void;
  getStart(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setStart(value?: google_protobuf_timestamp_pb.Timestamp): void;

  hasEnd(): boolean;
  clearEnd(): void;
  getEnd(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setEnd(value?: google_protobuf_timestamp_pb.Timestamp): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ValuesRequest.AsObject;
  static toObject(includeInstance: boolean, msg: ValuesRequest): ValuesRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ValuesRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ValuesRequest;
  static deserializeBinaryFromReader(message: ValuesRequest, reader: jspb.BinaryReader): ValuesRequest;
}

export namespace ValuesRequest {
  export type AsObject = {
    labelName: string,
    matchList: Array<string>,
    start?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    end?: google_protobuf_timestamp_pb.Timestamp.AsObject,
  }
}

export class ValuesResponse extends jspb.Message {
  clearLabelValuesList(): void;
  getLabelValuesList(): Array<string>;
  setLabelValuesList(value: Array<string>): void;
  addLabelValues(value: string, index?: number): string;

  clearWarningsList(): void;
  getWarningsList(): Array<string>;
  setWarningsList(value: Array<string>): void;
  addWarnings(value: string, index?: number): string;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ValuesResponse.AsObject;
  static toObject(includeInstance: boolean, msg: ValuesResponse): ValuesResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ValuesResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ValuesResponse;
  static deserializeBinaryFromReader(message: ValuesResponse, reader: jspb.BinaryReader): ValuesResponse;
}

export namespace ValuesResponse {
  export type AsObject = {
    labelValuesList: Array<string>,
    warningsList: Array<string>,
  }
}

export class ValueType extends jspb.Message {
  getType(): string;
  setType(value: string): void;

  getUnit(): string;
  setUnit(value: string): void;

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
    type: string,
    unit: string,
  }
}

