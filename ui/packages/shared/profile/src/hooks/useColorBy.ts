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

import {useCallback} from 'react';

import {useQueryState} from 'nuqs';

import {USER_PREFERENCES, useUserPreference} from '@parca/hooks';

import {stringParam} from './urlParsers';

export const useColorBy = (): {
  colorBy: string;
  setColorBy: (value: string) => void;
} => {
  const [colorByPreference, setColorByPreference] = useUserPreference<string>(
    USER_PREFERENCES.COLOR_BY.key
  );
  const [colorByRaw, setRawColorBy] = useQueryState('color_by', stringParam);

  const colorBy = colorByRaw ?? colorByPreference ?? 'binary';

  const setColorBy = useCallback(
    (value: string) => {
      void setRawColorBy(value);
      setColorByPreference(value);
    },
    [setRawColorBy, setColorByPreference]
  );

  return {colorBy, setColorBy};
};
