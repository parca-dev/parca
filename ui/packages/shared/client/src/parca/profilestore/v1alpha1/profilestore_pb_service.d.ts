// package: parca.profilestore.v1alpha1
// file: parca/profilestore/v1alpha1/profilestore.proto

import * as parca_profilestore_v1alpha1_profilestore_pb from "../../../parca/profilestore/v1alpha1/profilestore_pb";
import {grpc} from "@improbable-eng/grpc-web";

type ProfileStoreServiceWriteRaw = {
  readonly methodName: string;
  readonly service: typeof ProfileStoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof parca_profilestore_v1alpha1_profilestore_pb.WriteRawRequest;
  readonly responseType: typeof parca_profilestore_v1alpha1_profilestore_pb.WriteRawResponse;
};

export class ProfileStoreService {
  static readonly serviceName: string;
  static readonly WriteRaw: ProfileStoreServiceWriteRaw;
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

export class ProfileStoreServiceClient {
  readonly serviceHost: string;

  constructor(serviceHost: string, options?: grpc.RpcOptions);
  writeRaw(
    requestMessage: parca_profilestore_v1alpha1_profilestore_pb.WriteRawRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: parca_profilestore_v1alpha1_profilestore_pb.WriteRawResponse|null) => void
  ): UnaryResponse;
  writeRaw(
    requestMessage: parca_profilestore_v1alpha1_profilestore_pb.WriteRawRequest,
    callback: (error: ServiceError|null, responseMessage: parca_profilestore_v1alpha1_profilestore_pb.WriteRawResponse|null) => void
  ): UnaryResponse;
}

