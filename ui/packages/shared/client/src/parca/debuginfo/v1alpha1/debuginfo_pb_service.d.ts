// package: parca.debuginfo.v1alpha1
// file: parca/debuginfo/v1alpha1/debuginfo.proto

import * as parca_debuginfo_v1alpha1_debuginfo_pb from "../../../parca/debuginfo/v1alpha1/debuginfo_pb";
import {grpc} from "@improbable-eng/grpc-web";

type DebugInfoServiceExists = {
  readonly methodName: string;
  readonly service: typeof DebugInfoService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof parca_debuginfo_v1alpha1_debuginfo_pb.ExistsRequest;
  readonly responseType: typeof parca_debuginfo_v1alpha1_debuginfo_pb.ExistsResponse;
};

type DebugInfoServiceUpload = {
  readonly methodName: string;
  readonly service: typeof DebugInfoService;
  readonly requestStream: true;
  readonly responseStream: false;
  readonly requestType: typeof parca_debuginfo_v1alpha1_debuginfo_pb.UploadRequest;
  readonly responseType: typeof parca_debuginfo_v1alpha1_debuginfo_pb.UploadResponse;
};

export class DebugInfoService {
  static readonly serviceName: string;
  static readonly Exists: DebugInfoServiceExists;
  static readonly Upload: DebugInfoServiceUpload;
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

export class DebugInfoServiceClient {
  readonly serviceHost: string;

  constructor(serviceHost: string, options?: grpc.RpcOptions);
  exists(
    requestMessage: parca_debuginfo_v1alpha1_debuginfo_pb.ExistsRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: parca_debuginfo_v1alpha1_debuginfo_pb.ExistsResponse|null) => void
  ): UnaryResponse;
  exists(
    requestMessage: parca_debuginfo_v1alpha1_debuginfo_pb.ExistsRequest,
    callback: (error: ServiceError|null, responseMessage: parca_debuginfo_v1alpha1_debuginfo_pb.ExistsResponse|null) => void
  ): UnaryResponse;
  upload(metadata?: grpc.Metadata): RequestStream<parca_debuginfo_v1alpha1_debuginfo_pb.UploadRequest>;
}

