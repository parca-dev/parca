import cx from 'classnames';

const BUTTON_VARIANT = {
  primary: {
    text: 'text-gray-100 dark-gray-900 justify-center',
    bg: 'bg-indigo-600',
    border: 'border border-indigo-500',
    fontWeight: 'font-medium',
    hover: '',
    padding: 'py-2 px-4',
  },
  neutral: {
    text: 'text-gray-600 dark:text-gray-100 justify-center',
    bg: 'bg-gray-50 dark:bg-gray-900',
    border: 'border border-gray-200 dark:border-gray-600',
    fontWeight: 'font-normal',
    hover: '',
    padding: 'py-2 px-4',
  },
  link: {
    text: 'text-gray-600 dark:text-gray-300 justify-start',
    bg: '',
    border: '',
    fontWeight: 'font-normal',
    hover: 'hover:underline p-0',
    padding: 'py-1',
  },
};

export type ButtonVariant = keyof typeof BUTTON_VARIANT;

const Button = ({
  disabled = false,
  variant = 'primary',
  children,
  className = '',
  ...props
}: {
  disabled?: boolean;
  variant?: ButtonVariant;
  className?: string;
  children: React.ReactNode;
} & JSX.IntrinsicElements['button']) => {
  return (
    <button
      type="button"
      className={cx(
        disabled ? 'opacity-50 pointer-events-none' : '',
        ...Object.values(BUTTON_VARIANT[variant]),
        'cursor-pointer group relative w-full flex $ text-sm rounded-md text-whitefocus:outline-none focus:ring-2 focus:ring-offset-2',
        className
      )}
      disabled={disabled}
      {...props}
    >
      {children}
    </button>
  );
};

export default Button;
