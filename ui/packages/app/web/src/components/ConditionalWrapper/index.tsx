const ConditionalWrapper = ({condition, wrapper: Wrapper, children}) => {
  if (condition) {
    return <Wrapper>{children}</Wrapper>;
  }

  return children;
};

export default ConditionalWrapper;
