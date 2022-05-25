import {RefObject, useRef, useState, useEffect} from 'react';

export const useContainerDimensions = (): {
  dimensions?: DOMRect;
  ref: RefObject<HTMLDivElement>;
} => {
  const ref = useRef<HTMLDivElement>(null);
  const [dimensions, setDimensions] = useState<DOMRect>();

  const updateDimensions = () => setDimensions(ref.current?.getBoundingClientRect());

  useEffect(() => {
    updateDimensions();
    window.addEventListener('resize', updateDimensions);
    return () => window.removeEventListener('resize', updateDimensions);
  }, []);

  return {dimensions, ref};
};
