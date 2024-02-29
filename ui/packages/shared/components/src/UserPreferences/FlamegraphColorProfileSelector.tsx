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

import {useEffect} from 'react';

import {USER_PREFERENCES, useUserPreference} from '@parca/hooks';
import {selectColorProfiles, selectDarkMode, useAppSelector} from '@parca/store';
import {type ColorProfileName, type ColorsDuo} from '@parca/utilities';

const FlamegraphColorProfileSelector = (): JSX.Element => {
  const [colorProfileName, setColorProfileName] = useUserPreference<string>(
    USER_PREFERENCES.FLAMEGRAPH_COLOR_PROFILE.key
  );
  const isDarkMode = useAppSelector(selectDarkMode);
  const colorProfiles = useAppSelector(selectColorProfiles);

  useEffect(() => {
    if (colorProfiles[colorProfileName] === undefined) {
      // If the color profile is not found, set it to the default color profile
      setColorProfileName('ocean');
    }
  }, [colorProfiles, colorProfileName, setColorProfileName]);

  return (
    <fieldset className={'mt-2 flex items-center gap-2'}>
      <legend className="mb-1">Flamegraph Color Profile</legend>
      <div className="flex flex-col gap-3 pl-4 text-sm">
        {(Object.keys(colorProfiles) as ColorProfileName[]).map(profile => {
          const isSelected = profile === colorProfileName;
          let displayName: string = profile;
          if (profile === 'default') {
            displayName = 'neutral';
          }
          if (profile === 'ocean') {
            displayName = 'cool';
          }
          return (
            <div className="flex items-center gap-3" key={profile}>
              <label key={profile} className="flex items-center gap-1 capitalize">
                <input
                  type="radio"
                  name="colorProfile"
                  value={profile}
                  checked={isSelected}
                  onChange={e => {
                    if (e.target.checked) {
                      setColorProfileName(e.target.value);
                    }
                  }}
                />
                <span className="whitespace-nowrap">{displayName}</span>
              </label>
              <div className="flex w-72 gap-2 overflow-auto">
                {colorProfiles[profile].colors.map((color: ColorsDuo) => (
                  <div
                    key={`${color[0]}-${color[1]}`}
                    className="h-4 w-3 flex-shrink-0 rounded-full"
                    style={{backgroundColor: isDarkMode ? color[0] : color[1]}}
                  />
                ))}
              </div>
            </div>
          );
        })}
      </div>
    </fieldset>
  );
};

export default FlamegraphColorProfileSelector;
