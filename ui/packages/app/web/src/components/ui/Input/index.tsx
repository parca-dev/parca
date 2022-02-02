import cx from 'classnames';

const Input = ({className = '', ...props}) => {
  return (
    <input
      {...props}
      className={cx(
        'p-2 rounded-md bg-gray-50 dark:bg-gray-900 border border-gray-200 dark:border-gray-600',
        {
          [className]: !!className,
        }
      )}
    />
  );
};

export default Input;
