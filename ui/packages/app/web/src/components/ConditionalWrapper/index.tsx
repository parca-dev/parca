interface ConditionalWrapperProps {
  condition: boolean;
  wrapper: React.ComponentType<any>;
  children: React.ReactNode;
}

const ConditionalWrapper = ({condition, wrapper: Wrapper, children}: ConditionalWrapperProps) => {
  if (condition) {
    return <Wrapper>{children}</Wrapper>;
  }

  return <>{children}</>;
};

export default ConditionalWrapper;
