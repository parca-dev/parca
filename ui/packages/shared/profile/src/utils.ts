import {QueryRequest, QueryRequest_ReportType, QueryServiceClient} from '@parca/client';
import {RpcMetadata} from '@protobuf-ts/runtime-rpc';

export const hexifyAddress = (address?: string): string => {
  if (address == null) {
    return '';
  }
  return `0x${parseInt(address, 10).toString(16)}`;
};

export const downloadPprof = async (
  request: QueryRequest,
  queryClient: QueryServiceClient,
  metadata: RpcMetadata
) => {
  const req = {
    ...request,
    reportType: QueryRequest_ReportType.PPROF,
  };

  const {response} = await queryClient.query(req, {meta: metadata});
  if (response.report.oneofKind !== 'pprof') {
    throw new Error(
      `Expected pprof report, got: ${
        response.report.oneofKind !== undefined ? response.report.oneofKind : 'undefined'
      }`
    );
  }
  const blob = new Blob([response.report.pprof], {type: 'application/octet-stream'});
  return blob;
};
