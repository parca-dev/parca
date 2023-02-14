import commandLineArgs from 'command-line-args';

const optionDefinitions = [{name: 'apiEndpoint', type: String}];
const options = commandLineArgs(optionDefinitions);

export const getApiEndPoint = () => {
  return options.apiEndpoint ?? 'https://demo.parca.dev';
};

export const getGrpcMetadata = () => {
  return {meta: JSON.parse(process.env.GRPC_METADATA ?? '{}')};
};
