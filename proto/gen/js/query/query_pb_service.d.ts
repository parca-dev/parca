// package: parca.query
// file: query/query.proto

import * as query_query_pb from "../query/query_pb";
import {grpc} from "@improbable-eng/grpc-web";

type QueryQueryRange = {
  readonly methodName: string;
  readonly service: typeof Query;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof query_query_pb.QueryRangeRequest;
  readonly responseType: typeof query_query_pb.QueryRangeResponse;
};

type QueryQuery = {
  readonly methodName: string;
  readonly service: typeof Query;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof query_query_pb.QueryRequest;
  readonly responseType: typeof query_query_pb.QueryResponse;
};

type QuerySeries = {
  readonly methodName: string;
  readonly service: typeof Query;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof query_query_pb.SeriesRequest;
  readonly responseType: typeof query_query_pb.SeriesResponse;
};

type QueryLabels = {
  readonly methodName: string;
  readonly service: typeof Query;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof query_query_pb.LabelsRequest;
  readonly responseType: typeof query_query_pb.LabelsResponse;
};

type QueryValues = {
  readonly methodName: string;
  readonly service: typeof Query;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof query_query_pb.ValuesRequest;
  readonly responseType: typeof query_query_pb.ValuesResponse;
};

type QueryConfig = {
  readonly methodName: string;
  readonly service: typeof Query;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof query_query_pb.ConfigRequest;
  readonly responseType: typeof query_query_pb.ConfigResponse;
};

type QueryTargets = {
  readonly methodName: string;
  readonly service: typeof Query;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof query_query_pb.TargetsRequest;
  readonly responseType: typeof query_query_pb.TargetsResponse;
};

export class Query {
  static readonly serviceName: string;
  static readonly QueryRange: QueryQueryRange;
  static readonly Query: QueryQuery;
  static readonly Series: QuerySeries;
  static readonly Labels: QueryLabels;
  static readonly Values: QueryValues;
  static readonly Config: QueryConfig;
  static readonly Targets: QueryTargets;
}

export type ServiceError = { message: string, code: number; metadata: grpc.Metadata }
export type Status = { details: string, code: number; metadata: grpc.Metadata }

interface UnaryResponse {
  cancel(): void;
}
interface ResponseStream<T> {
  cancel(): void;
  on(type: 'data', handler: (message: T) => void): ResponseStream<T>;
  on(type: 'end', handler: (status?: Status) => void): ResponseStream<T>;
  on(type: 'status', handler: (status: Status) => void): ResponseStream<T>;
}
interface RequestStream<T> {
  write(message: T): RequestStream<T>;
  end(): void;
  cancel(): void;
  on(type: 'end', handler: (status?: Status) => void): RequestStream<T>;
  on(type: 'status', handler: (status: Status) => void): RequestStream<T>;
}
interface BidirectionalStream<ReqT, ResT> {
  write(message: ReqT): BidirectionalStream<ReqT, ResT>;
  end(): void;
  cancel(): void;
  on(type: 'data', handler: (message: ResT) => void): BidirectionalStream<ReqT, ResT>;
  on(type: 'end', handler: (status?: Status) => void): BidirectionalStream<ReqT, ResT>;
  on(type: 'status', handler: (status: Status) => void): BidirectionalStream<ReqT, ResT>;
}

export class QueryClient {
  readonly serviceHost: string;

  constructor(serviceHost: string, options?: grpc.RpcOptions);
  queryRange(
    requestMessage: query_query_pb.QueryRangeRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: query_query_pb.QueryRangeResponse|null) => void
  ): UnaryResponse;
  queryRange(
    requestMessage: query_query_pb.QueryRangeRequest,
    callback: (error: ServiceError|null, responseMessage: query_query_pb.QueryRangeResponse|null) => void
  ): UnaryResponse;
  query(
    requestMessage: query_query_pb.QueryRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: query_query_pb.QueryResponse|null) => void
  ): UnaryResponse;
  query(
    requestMessage: query_query_pb.QueryRequest,
    callback: (error: ServiceError|null, responseMessage: query_query_pb.QueryResponse|null) => void
  ): UnaryResponse;
  series(
    requestMessage: query_query_pb.SeriesRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: query_query_pb.SeriesResponse|null) => void
  ): UnaryResponse;
  series(
    requestMessage: query_query_pb.SeriesRequest,
    callback: (error: ServiceError|null, responseMessage: query_query_pb.SeriesResponse|null) => void
  ): UnaryResponse;
  labels(
    requestMessage: query_query_pb.LabelsRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: query_query_pb.LabelsResponse|null) => void
  ): UnaryResponse;
  labels(
    requestMessage: query_query_pb.LabelsRequest,
    callback: (error: ServiceError|null, responseMessage: query_query_pb.LabelsResponse|null) => void
  ): UnaryResponse;
  values(
    requestMessage: query_query_pb.ValuesRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: query_query_pb.ValuesResponse|null) => void
  ): UnaryResponse;
  values(
    requestMessage: query_query_pb.ValuesRequest,
    callback: (error: ServiceError|null, responseMessage: query_query_pb.ValuesResponse|null) => void
  ): UnaryResponse;
  config(
    requestMessage: query_query_pb.ConfigRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: query_query_pb.ConfigResponse|null) => void
  ): UnaryResponse;
  config(
    requestMessage: query_query_pb.ConfigRequest,
    callback: (error: ServiceError|null, responseMessage: query_query_pb.ConfigResponse|null) => void
  ): UnaryResponse;
  targets(
    requestMessage: query_query_pb.TargetsRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: query_query_pb.TargetsResponse|null) => void
  ): UnaryResponse;
  targets(
    requestMessage: query_query_pb.TargetsRequest,
    callback: (error: ServiceError|null, responseMessage: query_query_pb.TargetsResponse|null) => void
  ): UnaryResponse;
}

