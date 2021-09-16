import cx from 'classnames'

const BUTTON_COLORS = {
  primary: {
    text: 'text-gray-100 dark-gray-900',
    bg: 'bg-indigo-600',
    border: 'border-indigo-500'
  },
  neutral: {
    text: 'text-gray-900 dark:text-gray-100',
    bg: 'bg-gray-50 dark:bg-gray-900',
    border: 'border-gray-200 dark:border-gray-600'
  }
}

export type ButtonColor = keyof typeof BUTTON_COLORS

const Button = ({
  disabled = false,
  color = 'primary',
  children,
  ...props
}: {
  disabled?: boolean
  color?: ButtonColor
  children: React.ReactNode
} & JSX.IntrinsicElements['button']) => {
  return (
    <button
      type='button'
      className={cx(
        disabled ? 'opacity-50 pointer-events-none' : '',
        BUTTON_COLORS[color].bg,
        BUTTON_COLORS[color].text,
        `cursor-pointer group relative w-full flex justify-center py-2 px-4 border-t border-r border-b border-l ${BUTTON_COLORS[color].border} text-sm font-medium rounded-md text-whitefocus:outline-none focus:ring-2 focus:ring-offset-2`
      )}
      disabled={disabled}
      {...props}
    >
      {children}
    </button>
  )
}

export default Button
