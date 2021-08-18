// package: parca.profilestore
// file: profilestore/profilestore.proto

import * as profilestore_profilestore_pb from "../profilestore/profilestore_pb";
import {grpc} from "@improbable-eng/grpc-web";

type ProfileStoreWriteRaw = {
  readonly methodName: string;
  readonly service: typeof ProfileStore;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof profilestore_profilestore_pb.WriteRawRequest;
  readonly responseType: typeof profilestore_profilestore_pb.WriteRawResponse;
};

export class ProfileStore {
  static readonly serviceName: string;
  static readonly WriteRaw: ProfileStoreWriteRaw;
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

export class ProfileStoreClient {
  readonly serviceHost: string;

  constructor(serviceHost: string, options?: grpc.RpcOptions);
  writeRaw(
    requestMessage: profilestore_profilestore_pb.WriteRawRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: profilestore_profilestore_pb.WriteRawResponse|null) => void
  ): UnaryResponse;
  writeRaw(
    requestMessage: profilestore_profilestore_pb.WriteRawRequest,
    callback: (error: ServiceError|null, responseMessage: profilestore_profilestore_pb.WriteRawResponse|null) => void
  ): UnaryResponse;
}

