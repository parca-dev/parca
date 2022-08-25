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

/* eslint-disable */
import useLocalStorageState from 'use-local-storage-state';

const UI_FLAGS = 'ui-flags';

const initializeFlagsFromURL = () => {
  if (typeof window === 'undefined') {
    return;
  }

  const url = new URL(window.location.href);
  const enableFlag = url.searchParams.get('enable-ui-flag');
  const disableFlag = url.searchParams.get('disable-ui-flag');
  if (!enableFlag && !disableFlag) {
    return;
  }
  const flags = JSON.parse(window.localStorage.getItem(UI_FLAGS) ?? '{}');
  if (enableFlag) {
    flags[enableFlag] = true;
  }
  if (disableFlag) {
    delete flags[disableFlag];
  }
  window.localStorage.setItem(UI_FLAGS, JSON.stringify(flags));
};

initializeFlagsFromURL();

const useUIFeatureFlag = (featureFlag: string, defaultValue: boolean = false) => {
  const [flags, setFlags] = useLocalStorageState(UI_FLAGS, {
    defaultValue: {},
  });

  const value = flags[featureFlag] || defaultValue;
  const setFlag = (flag: boolean) => {
    setFlags({...flags, [featureFlag]: flag});
  };

  return [value, setFlag];
};

export default useUIFeatureFlag;
