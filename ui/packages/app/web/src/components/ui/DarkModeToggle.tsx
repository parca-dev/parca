import {MoonIcon, SunIcon} from '@heroicons/react/solid';
import {useAppSelector, useAppDispatch} from '../../store/hooks';
import {selectDarkMode, setDarkMode} from '../../store/slices/uiSlice';

const DarkModeToggle = () => {
  const dispatch = useAppDispatch();
  const isDarkMode = useAppSelector(selectDarkMode);
  const Icon = isDarkMode ? MoonIcon : SunIcon;
  return (
    <button
      type="button"
      className="p-1 cursor-pointer rounded-full flex align-center items-center"
      onClick={() => {}}
    >
      <Icon
        onClick={() => dispatch(setDarkMode(!isDarkMode))}
        className="w-5 h-5"
        aria-hidden="true"
      />
    </button>
  );
};

export default DarkModeToggle;
