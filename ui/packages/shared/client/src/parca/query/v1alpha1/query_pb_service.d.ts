// package: parca.query.v1alpha1
// file: parca/query/v1alpha1/query.proto

import * as parca_query_v1alpha1_query_pb from "../../../parca/query/v1alpha1/query_pb";
import {grpc} from "@improbable-eng/grpc-web";

type QueryServiceQueryRange = {
  readonly methodName: string;
  readonly service: typeof QueryService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof parca_query_v1alpha1_query_pb.QueryRangeRequest;
  readonly responseType: typeof parca_query_v1alpha1_query_pb.QueryRangeResponse;
};

type QueryServiceQuery = {
  readonly methodName: string;
  readonly service: typeof QueryService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof parca_query_v1alpha1_query_pb.QueryRequest;
  readonly responseType: typeof parca_query_v1alpha1_query_pb.QueryResponse;
};

type QueryServiceSeries = {
  readonly methodName: string;
  readonly service: typeof QueryService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof parca_query_v1alpha1_query_pb.SeriesRequest;
  readonly responseType: typeof parca_query_v1alpha1_query_pb.SeriesResponse;
};

type QueryServiceLabels = {
  readonly methodName: string;
  readonly service: typeof QueryService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof parca_query_v1alpha1_query_pb.LabelsRequest;
  readonly responseType: typeof parca_query_v1alpha1_query_pb.LabelsResponse;
};

type QueryServiceValues = {
  readonly methodName: string;
  readonly service: typeof QueryService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof parca_query_v1alpha1_query_pb.ValuesRequest;
  readonly responseType: typeof parca_query_v1alpha1_query_pb.ValuesResponse;
};

export class QueryService {
  static readonly serviceName: string;
  static readonly QueryRange: QueryServiceQueryRange;
  static readonly Query: QueryServiceQuery;
  static readonly Series: QueryServiceSeries;
  static readonly Labels: QueryServiceLabels;
  static readonly Values: QueryServiceValues;
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

export class QueryServiceClient {
  readonly serviceHost: string;

  constructor(serviceHost: string, options?: grpc.RpcOptions);
  queryRange(
    requestMessage: parca_query_v1alpha1_query_pb.QueryRangeRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: parca_query_v1alpha1_query_pb.QueryRangeResponse|null) => void
  ): UnaryResponse;
  queryRange(
    requestMessage: parca_query_v1alpha1_query_pb.QueryRangeRequest,
    callback: (error: ServiceError|null, responseMessage: parca_query_v1alpha1_query_pb.QueryRangeResponse|null) => void
  ): UnaryResponse;
  query(
    requestMessage: parca_query_v1alpha1_query_pb.QueryRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: parca_query_v1alpha1_query_pb.QueryResponse|null) => void
  ): UnaryResponse;
  query(
    requestMessage: parca_query_v1alpha1_query_pb.QueryRequest,
    callback: (error: ServiceError|null, responseMessage: parca_query_v1alpha1_query_pb.QueryResponse|null) => void
  ): UnaryResponse;
  series(
    requestMessage: parca_query_v1alpha1_query_pb.SeriesRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: parca_query_v1alpha1_query_pb.SeriesResponse|null) => void
  ): UnaryResponse;
  series(
    requestMessage: parca_query_v1alpha1_query_pb.SeriesRequest,
    callback: (error: ServiceError|null, responseMessage: parca_query_v1alpha1_query_pb.SeriesResponse|null) => void
  ): UnaryResponse;
  labels(
    requestMessage: parca_query_v1alpha1_query_pb.LabelsRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: parca_query_v1alpha1_query_pb.LabelsResponse|null) => void
  ): UnaryResponse;
  labels(
    requestMessage: parca_query_v1alpha1_query_pb.LabelsRequest,
    callback: (error: ServiceError|null, responseMessage: parca_query_v1alpha1_query_pb.LabelsResponse|null) => void
  ): UnaryResponse;
  values(
    requestMessage: parca_query_v1alpha1_query_pb.ValuesRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: parca_query_v1alpha1_query_pb.ValuesResponse|null) => void
  ): UnaryResponse;
  values(
    requestMessage: parca_query_v1alpha1_query_pb.ValuesRequest,
    callback: (error: ServiceError|null, responseMessage: parca_query_v1alpha1_query_pb.ValuesResponse|null) => void
  ): UnaryResponse;
}

