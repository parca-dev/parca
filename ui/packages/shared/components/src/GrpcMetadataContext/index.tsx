import {grpc} from '@improbable-eng/grpc-web';
import {createContext, ReactNode, useContext} from 'react';

const EMPTY_METADATA = new grpc.Metadata();
const GrpcMetadataContext = createContext<grpc.Metadata>(EMPTY_METADATA);

export const GrpcMetadataProvider = ({
  children,
  value,
}: {
  children: ReactNode;
  value?: grpc.Metadata;
}) => {
  return (
    <GrpcMetadataContext.Provider value={value ?? EMPTY_METADATA}>
      {children}
    </GrpcMetadataContext.Provider>
  );
};

export const useGrpcMetadata = () => {
  const context = useContext(GrpcMetadataContext);
  if (context == null) {
    return EMPTY_METADATA;
  }
  return context;
};

export default GrpcMetadataContext;
