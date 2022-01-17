const Input = ({className, ...props}) => {
  return <input {...props} className={`p-1 rounded-sm ${className || ''}`} />;
};

export default Input;
