import cx from 'classnames'

export type ButtonColor = 'primary' | 'secondary' | 'neutral'

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
        `bg-${color}`,
        'cursor-pointer group relative w-full flex justify-center py-2 px-4 border border-transparent text-sm font-medium rounded-md text-whitefocus:outline-none focus:ring-2 focus:ring-offset-2'
      )}
      disabled={disabled}
      {...props}
    >
      {children}
    </button>
  )
}

export default Button
