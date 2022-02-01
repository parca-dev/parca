import cx from 'classnames';

const VARIANTS = {
  primary: {
    color: 'text-gray-600',
    bg: 'bg-indigo-100',
  },
  success: {
    color: 'text-green-800',
    bg: 'bg-green-100',
  },
  danger: {
    color: 'text-red-800',
    bg: 'bg-red-100',
  },
  warning: {
    color: 'text-amber-800',
    bg: 'bg-amber-100',
  },
  info: {
    color: 'text-blue-600',
    bg: 'bg-blue-100',
  },
  neutral: {
    color: 'text-neutral-800',
    bg: 'bg-neutral-100',
  },
};

export type Variant = keyof typeof VARIANTS;

const Pill = ({
  inverted = false,
  variant = 'primary',
  children,
  ...props
}: {
  inverted?: boolean;
  variant?: Variant;
  children: React.ReactNode;
} & JSX.IntrinsicElements['span']) => (
  <span
    className={cx(
      VARIANTS[variant].color,
      VARIANTS[variant].bg,
      `px-2 inline-flex text-xs leading-5 font-semibold rounded-full whitespace-nowrap `
    )}
    {...props}
  >
    {children}
  </span>
);

export default Pill;
