const Input = ({className = '', ...props}) => {
  return (
    <input {...props} className={`p-1 rounded-sm ${className?.length > 0 ? className : ''}`} />
  );
};

export default Input;
