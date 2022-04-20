import {RpcMetadata} from '@protobuf-ts/runtime-rpc';
import {createContext, ReactNode, useContext} from 'react';

const GrpcMetadataContext = createContext<RpcMetadata>({});

export const GrpcMetadataProvider = ({
  children,
  value,
}: {
  children: ReactNode;
  value?: RpcMetadata;
}) => {
  return (
    <GrpcMetadataContext.Provider value={value ?? {}}>{children}</GrpcMetadataContext.Provider>
  );
};

export const useGrpcMetadata = () => {
  const context = useContext(GrpcMetadataContext);
  if (context == null) {
    return {};
  }
  return context;
};

export default GrpcMetadataContext;
