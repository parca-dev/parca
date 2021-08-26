// package: parca.scrape
// file: scrape/scrape.proto

import * as scrape_scrape_pb from "../scrape/scrape_pb";
import {grpc} from "@improbable-eng/grpc-web";

type ScrapeTargets = {
  readonly methodName: string;
  readonly service: typeof Scrape;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof scrape_scrape_pb.TargetsRequest;
  readonly responseType: typeof scrape_scrape_pb.TargetsResponse;
};

export class Scrape {
  static readonly serviceName: string;
  static readonly Targets: ScrapeTargets;
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

export class ScrapeClient {
  readonly serviceHost: string;

  constructor(serviceHost: string, options?: grpc.RpcOptions);
  targets(
    requestMessage: scrape_scrape_pb.TargetsRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: scrape_scrape_pb.TargetsResponse|null) => void
  ): UnaryResponse;
  targets(
    requestMessage: scrape_scrape_pb.TargetsRequest,
    callback: (error: ServiceError|null, responseMessage: scrape_scrape_pb.TargetsResponse|null) => void
  ): UnaryResponse;
}

