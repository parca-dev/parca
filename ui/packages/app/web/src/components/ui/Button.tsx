import cx from 'classnames';

const BUTTON_COLORS = {
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

export type ButtonColor = keyof typeof BUTTON_COLORS;

const Button = ({
  disabled = false,
  color = 'primary',
  children,
  additionalClasses,
  ...props
}: {
  disabled?: boolean;
  color?: ButtonColor;
  additionalClasses?: string;
  children: React.ReactNode;
} & JSX.IntrinsicElements['button']) => {
  return (
    <button
      type="button"
      className={cx(
        disabled ? 'opacity-50 pointer-events-none' : '',
        BUTTON_COLORS[color].bg,
        BUTTON_COLORS[color].text,
        /* eslint-disable @typescript-eslint/restrict-template-expressions */
        `cursor-pointer group relative w-full flex ${BUTTON_COLORS[color].padding} ${BUTTON_COLORS[color].border} text-sm ${BUTTON_COLORS[color].fontWeight} rounded-md text-whitefocus:outline-none focus:ring-2 focus:ring-offset-2 ${BUTTON_COLORS[color].hover} ${additionalClasses}`
      )}
      disabled={disabled}
      {...props}
    >
      {children}
    </button>
  );
};

export default Button;
