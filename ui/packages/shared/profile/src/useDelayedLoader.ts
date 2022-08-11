import {useEffect, useState} from 'react';

interface DelayedLoaderOptions {
  delay?: number;
}

const useDelayedLoader = (isLoading = false, options?: DelayedLoaderOptions) => {
  const {delay = 500} = options != null || {};
  const [isLoaderVisible, setIsLoaderVisible] = useState<boolean>(false);
  useEffect(() => {
    let showLoaderTimeout;
    if (isLoading && !isLoaderVisible) {
      // if the request takes longer than half a second, show the loading icon
      showLoaderTimeout = setTimeout(() => {
        setIsLoaderVisible(true);
      }, delay);
    } else {
      setIsLoaderVisible(false);
    }
    return () => clearTimeout(showLoaderTimeout);
  }, [isLoading]);

  return isLoaderVisible;
};

export default useDelayedLoader;
