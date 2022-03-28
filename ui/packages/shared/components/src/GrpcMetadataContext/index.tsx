import {grpc} from '@improbable-eng/grpc-web';
import React from 'react';

const EMPTY_METADATA = new grpc.Metadata();
const GrpcMetadataContext = React.createContext<grpc.Metadata>(EMPTY_METADATA);

export const GrpcMetadataProvider = ({
  children,
  value,
}: {
  children: React.ReactNode;
  value?: grpc.Metadata;
}) => {
  return (
    <GrpcMetadataContext.Provider value={value || EMPTY_METADATA}>
      {children}
    </GrpcMetadataContext.Provider>
  );
};

export const useGrpcMetadata = () => {
  const context = React.useContext(GrpcMetadataContext);
  if (!context) {
    return EMPTY_METADATA;
  }
  return context;
};

export default GrpcMetadataContext;
