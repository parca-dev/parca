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
  default: boolean | string;
}

export const USER_PREFERENCES: {[key: string]: UserPreferenceDetails} = {
  HIGHTLIGHT_AFTER_FILTERING: {
    name: 'Highlight matching nodes after filtering',
    key: 'HIGHTLIGHT_AFTER_FILTERING',
    type: 'boolean',
    default: true,
  },
  ENABLE_GRAPH_TRIMMING: {
    name: 'Enable trimming of nodes that are too small to be seen',
    key: 'ENABLE_GRAPH_TRIMMING',
    type: 'boolean',
    default: true,
  },
  FLAMEGRAPH_COLOR_PROFILE: {
    name: 'Flamegraph color profile',
    key: 'FLAMEGRAPH_COLOR_PROFILE',
    type: 'string',
    default: 'default',
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
