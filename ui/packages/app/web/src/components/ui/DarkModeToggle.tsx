import { MoonIcon, SunIcon } from '@heroicons/react/solid'

const DarkModeToggle = ({
  isDarkMode,
  setDarkMode
}: {
  isDarkMode: boolean
  setDarkMode: (mode: boolean) => void
}) => {
  const Icon = isDarkMode ? MoonIcon : SunIcon
  return (
    <button
      type='button'
      className='p-1 cursor-pointer rounded-full flex align-center items-center'
      onClick={() => {}}
    >
      <Icon onClick={() => setDarkMode(!isDarkMode)} className='w-5 h-5' aria-hidden='true' />
    </button>
  )
}

export default DarkModeToggle
