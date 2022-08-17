// Copyright 2022 The Parca Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
