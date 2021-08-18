// package: parca.debuginfo
// file: debuginfo/debuginfo.proto

import * as debuginfo_debuginfo_pb from "../debuginfo/debuginfo_pb";
import {grpc} from "@improbable-eng/grpc-web";

type DebugInfoExists = {
  readonly methodName: string;
  readonly service: typeof DebugInfo;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof debuginfo_debuginfo_pb.DebugInfoExistsRequest;
  readonly responseType: typeof debuginfo_debuginfo_pb.DebugInfoExistsResponse;
};

type DebugInfoUpload = {
  readonly methodName: string;
  readonly service: typeof DebugInfo;
  readonly requestStream: true;
  readonly responseStream: false;
  readonly requestType: typeof debuginfo_debuginfo_pb.DebugInfoUploadRequest;
  readonly responseType: typeof debuginfo_debuginfo_pb.DebugInfoUploadResponse;
};

export class DebugInfo {
  static readonly serviceName: string;
  static readonly Exists: DebugInfoExists;
  static readonly Upload: DebugInfoUpload;
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

export class DebugInfoClient {
  readonly serviceHost: string;

  constructor(serviceHost: string, options?: grpc.RpcOptions);
  exists(
    requestMessage: debuginfo_debuginfo_pb.DebugInfoExistsRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: debuginfo_debuginfo_pb.DebugInfoExistsResponse|null) => void
  ): UnaryResponse;
  exists(
    requestMessage: debuginfo_debuginfo_pb.DebugInfoExistsRequest,
    callback: (error: ServiceError|null, responseMessage: debuginfo_debuginfo_pb.DebugInfoExistsResponse|null) => void
  ): UnaryResponse;
  upload(metadata?: grpc.Metadata): RequestStream<debuginfo_debuginfo_pb.DebugInfoUploadRequest>;
}

