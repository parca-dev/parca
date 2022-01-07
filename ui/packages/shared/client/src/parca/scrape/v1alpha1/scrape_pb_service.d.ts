// package: parca.scrape.v1alpha1
// file: parca/scrape/v1alpha1/scrape.proto

import * as parca_scrape_v1alpha1_scrape_pb from "../../../parca/scrape/v1alpha1/scrape_pb";
import {grpc} from "@improbable-eng/grpc-web";

type ScrapeServiceTargets = {
  readonly methodName: string;
  readonly service: typeof ScrapeService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof parca_scrape_v1alpha1_scrape_pb.TargetsRequest;
  readonly responseType: typeof parca_scrape_v1alpha1_scrape_pb.TargetsResponse;
};

export class ScrapeService {
  static readonly serviceName: string;
  static readonly Targets: ScrapeServiceTargets;
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

export class ScrapeServiceClient {
  readonly serviceHost: string;

  constructor(serviceHost: string, options?: grpc.RpcOptions);
  targets(
    requestMessage: parca_scrape_v1alpha1_scrape_pb.TargetsRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: parca_scrape_v1alpha1_scrape_pb.TargetsResponse|null) => void
  ): UnaryResponse;
  targets(
    requestMessage: parca_scrape_v1alpha1_scrape_pb.TargetsRequest,
    callback: (error: ServiceError|null, responseMessage: parca_scrape_v1alpha1_scrape_pb.TargetsResponse|null) => void
  ): UnaryResponse;
}

