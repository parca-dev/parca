// package: parca.query
// file: query/query.proto

import * as jspb from "google-protobuf";
import * as google_api_annotations_pb from "../google/api/annotations_pb";
import * as google_protobuf_timestamp_pb from "google-protobuf/google/protobuf/timestamp_pb";
import * as google_pprof_profile_pb from "../google/pprof/profile_pb";
import * as profilestore_profilestore_pb from "../profilestore/profilestore_pb";

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
  getLabelset(): profilestore_profilestore_pb.LabelSet | undefined;
  setLabelset(value?: profilestore_profilestore_pb.LabelSet): void;

  clearSamplesList(): void;
  getSamplesList(): Array<MetricsSample>;
  setSamplesList(value: Array<MetricsSample>): void;
  addSamples(value?: MetricsSample, index?: number): MetricsSample;

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
    labelset?: profilestore_profilestore_pb.LabelSet.AsObject,
    samplesList: Array<MetricsSample.AsObject>,
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

export class QueryRequest extends jspb.Message {
  getMode(): QueryRequest.ModeMap[keyof QueryRequest.ModeMap];
  setMode(value: QueryRequest.ModeMap[keyof QueryRequest.ModeMap]): void;

  hasDiff(): boolean;
  clearDiff(): void;
  getDiff(): QueryRequest.Diff | undefined;
  setDiff(value?: QueryRequest.Diff): void;

  hasMerge(): boolean;
  clearMerge(): void;
  getMerge(): QueryRequest.Merge | undefined;
  setMerge(value?: QueryRequest.Merge): void;

  hasSingle(): boolean;
  clearSingle(): void;
  getSingle(): QueryRequest.Single | undefined;
  setSingle(value?: QueryRequest.Single): void;

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
    diff?: QueryRequest.Diff.AsObject,
    merge?: QueryRequest.Merge.AsObject,
    single?: QueryRequest.Single.AsObject,
    reportType: QueryRequest.ReportTypeMap[keyof QueryRequest.ReportTypeMap],
  }

  export class Diff extends jspb.Message {
    hasA(): boolean;
    clearA(): void;
    getA(): QueryRequest.ProfileSelect | undefined;
    setA(value?: QueryRequest.ProfileSelect): void;

    hasB(): boolean;
    clearB(): void;
    getB(): QueryRequest.ProfileSelect | undefined;
    setB(value?: QueryRequest.ProfileSelect): void;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): Diff.AsObject;
    static toObject(includeInstance: boolean, msg: Diff): Diff.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: Diff, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): Diff;
    static deserializeBinaryFromReader(message: Diff, reader: jspb.BinaryReader): Diff;
  }

  export namespace Diff {
    export type AsObject = {
      a?: QueryRequest.ProfileSelect.AsObject,
      b?: QueryRequest.ProfileSelect.AsObject,
    }
  }

  export class ProfileSelect extends jspb.Message {
    getMode(): string;
    setMode(value: string): void;

    hasTime(): boolean;
    clearTime(): void;
    getTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
    setTime(value?: google_protobuf_timestamp_pb.Timestamp): void;

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
    toObject(includeInstance?: boolean): ProfileSelect.AsObject;
    static toObject(includeInstance: boolean, msg: ProfileSelect): ProfileSelect.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: ProfileSelect, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): ProfileSelect;
    static deserializeBinaryFromReader(message: ProfileSelect, reader: jspb.BinaryReader): ProfileSelect;
  }

  export namespace ProfileSelect {
    export type AsObject = {
      mode: string,
      time?: google_protobuf_timestamp_pb.Timestamp.AsObject,
      query: string,
      start?: google_protobuf_timestamp_pb.Timestamp.AsObject,
      end?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    }
  }

  export class Merge extends jspb.Message {
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
    toObject(includeInstance?: boolean): Merge.AsObject;
    static toObject(includeInstance: boolean, msg: Merge): Merge.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: Merge, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): Merge;
    static deserializeBinaryFromReader(message: Merge, reader: jspb.BinaryReader): Merge;
  }

  export namespace Merge {
    export type AsObject = {
      query: string,
      start?: google_protobuf_timestamp_pb.Timestamp.AsObject,
      end?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    }
  }

  export class Single extends jspb.Message {
    hasTime(): boolean;
    clearTime(): void;
    getTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
    setTime(value?: google_protobuf_timestamp_pb.Timestamp): void;

    getQuery(): string;
    setQuery(value: string): void;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): Single.AsObject;
    static toObject(includeInstance: boolean, msg: Single): Single.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: Single, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): Single;
    static deserializeBinaryFromReader(message: Single, reader: jspb.BinaryReader): Single;
  }

  export namespace Single {
    export type AsObject = {
      time?: google_protobuf_timestamp_pb.Timestamp.AsObject,
      query: string,
    }
  }

  export interface ModeMap {
    SINGLE: 0;
    DIFF: 1;
    MERGE: 2;
  }

  export const Mode: ModeMap;

  export interface ReportTypeMap {
    FLAMEGRAPH: 0;
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
  getRoot(): FlamegraphNode | undefined;
  setRoot(value?: FlamegraphNode): void;

  getTotal(): number;
  setTotal(value: number): void;

  getUnit(): string;
  setUnit(value: string): void;

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
    root?: FlamegraphNode.AsObject,
    total: number,
    unit: string,
  }
}

export class FlamegraphNode extends jspb.Message {
  getName(): string;
  setName(value: string): void;

  getFullName(): string;
  setFullName(value: string): void;

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
    name: string,
    fullName: string,
    cumulative: number,
    diff: number,
    childrenList: Array<FlamegraphNode.AsObject>,
  }
}

export class QueryResponse extends jspb.Message {
  hasFlamegraph(): boolean;
  clearFlamegraph(): void;
  getFlamegraph(): Flamegraph | undefined;
  setFlamegraph(value?: Flamegraph): void;

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
  }

  export enum ReportCase {
    REPORT_NOT_SET = 0,
    FLAMEGRAPH = 5,
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
  }
}

export class ConfigRequest extends jspb.Message {
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ConfigRequest.AsObject;
  static toObject(includeInstance: boolean, msg: ConfigRequest): ConfigRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ConfigRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ConfigRequest;
  static deserializeBinaryFromReader(message: ConfigRequest, reader: jspb.BinaryReader): ConfigRequest;
}

export namespace ConfigRequest {
  export type AsObject = {
  }
}

export class ConfigResponse extends jspb.Message {
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ConfigResponse.AsObject;
  static toObject(includeInstance: boolean, msg: ConfigResponse): ConfigResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ConfigResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ConfigResponse;
  static deserializeBinaryFromReader(message: ConfigResponse, reader: jspb.BinaryReader): ConfigResponse;
}

export namespace ConfigResponse {
  export type AsObject = {
  }
}

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

