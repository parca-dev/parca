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

import {useLocalStorage} from '@rehooks/local-storage';

const USER_PREFERENCES_KEY = 'user-preferences';

export interface UserPreferenceDetails {
  name: string;
  key: string;
  type: string;
  default: any;
}

export const USER_PREFERENCES: {[key: string]: UserPreferenceDetails} = {
  HIGHTLIGHT_AFTER_FILTERING: {
    name: 'Highlight matching nodes after filtering',
    key: 'HIGHTLIGHT_AFTER_FILTERING',
    type: 'boolean',
    default: true,
  },
  DISABLE_GRAPH_TRIMMING: {
    name: 'Disable trimming of nodes that are too small to be seen',
    key: 'DISABLE_GRAPH_TRIMMING',
    type: 'boolean',
    default: false,
  },
} as const;

export type UserPreference = keyof typeof USER_PREFERENCES;

const useUserPreference = <T>(preferenceName: UserPreference): [T, (flag: T) => void] => {
  const [flags, setFlags] = useLocalStorage<{[flag: string]: any}>(USER_PREFERENCES_KEY, {});

  const value: T = flags[preferenceName] ?? USER_PREFERENCES[preferenceName].default;
  const setFlag = (flag: T): void => {
    setFlags({...flags, [preferenceName]: flag});
  };

  return [value, setFlag];
};

export default useUserPreference;
