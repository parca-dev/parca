const Card = ({ children }) => {
  return (
    <div className='mx-auto'>
      <div className='bg-white shadow overflow-hidden sm:rounded-lg flex-1 flex-column'>
        {children}
      </div>
    </div>
  )
}

const Header = ({ children }) => {
  return (
    <div
      className='bg-gray-100 px-4 py-4 sm:px-6'
      style={{ justifyContent: 'space-between', alignItems: 'stretch' }}
    >
      {children}
    </div>
  )
}

const Body = ({ children }) => {
  return <div className='border-t border-gray-200 p-4'>{children}</div>
}

export default Object.assign(Card, {
  Header,
  Body
})
