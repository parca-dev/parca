import cx from 'classnames';

const Input = ({className = '', ...props}) => {
  return <input {...props} className={cx('p-2 rounded-md', {[className]: !!className})} />;
};

export default Input;
