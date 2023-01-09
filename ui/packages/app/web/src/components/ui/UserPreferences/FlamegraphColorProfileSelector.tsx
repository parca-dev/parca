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

import {COLOR_PROFILES} from '@parca/functions';
import useUserPreference, {USER_PREFERENCES} from '@parca/functions/useUserPreference';
import {selectDarkMode, useAppSelector} from '@parca/store';

const FlamegraphColorProfileSelector = () => {
  const [colorProfileName, setColorProfileName] = useUserPreference<string>(
    USER_PREFERENCES.FLAMEGRAPH_COLOR_PROFILE.key
  );
  const isDarkMode = useAppSelector(selectDarkMode);

  return (
    <fieldset className={'flex gap-2 items-center mt-2'}>
      <legend className="mb-1">Flamegraph Color Profile</legend>
      <div className="flex flex-col pl-4 gap-2 text-sm">
        {Object.keys(COLOR_PROFILES).map(profile => {
          const isSelected = profile === colorProfileName;
          return (
            <div className="flex items-center">
              <label key={profile} className="capitalize flex gap-1 items-center w-24">
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
                <span>{profile}</span>
              </label>
              <div className="flex gap-2 flex-wrap w-64">
                {COLOR_PROFILES[profile].colors.map(color => (
                  <div
                    key={color}
                    className="w-4 h-4 rounded-full"
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
