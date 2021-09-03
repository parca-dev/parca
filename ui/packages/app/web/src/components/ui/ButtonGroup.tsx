const ButtonGroup = ({
  children,
  ...props
}: { children: React.ReactNode } & JSX.IntrinsicElements['div']) => {
  return (
    <div className='flex justify-center items-baseline flex-wrap' {...props}>
      <div className='flex space-x-1'>{children}</div>
    </div>
  )
}

export default ButtonGroup
