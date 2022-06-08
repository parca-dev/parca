import {useEffect, useState} from 'react';

const useIsShiftDown = () => {
  const [isShiftDown, setIsShiftDown] = useState(false);

  useEffect(() => {
    const handleShiftDown = (event: {keyCode: number}) => {
      if (event.keyCode === 16) {
        setIsShiftDown(true);
      }
    };
    window.addEventListener('keydown', handleShiftDown);
    const handleShiftUp = (event: {keyCode: number}) => {
      if (event.keyCode === 16) {
        setIsShiftDown(false);
      }
    };
    window.addEventListener('keyup', handleShiftUp);

    return () => {
      window.removeEventListener('keydown', handleShiftDown);
      window.removeEventListener('keyup', handleShiftUp);
    };
  }, []);

  return isShiftDown;
};

export default useIsShiftDown;
