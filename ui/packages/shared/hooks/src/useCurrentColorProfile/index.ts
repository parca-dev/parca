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

import {selectColorProfiles, useAppSelector} from '@parca/store';
import type {ColorConfig, ColorProfileName} from '@parca/utilities';

import useUserPreference, {USER_PREFERENCES} from '../useUserPreference';

const useCurrentColorProfile = (): ColorConfig => {
  const colorProfiles = useAppSelector(selectColorProfiles);
  const [colorProfile] = useUserPreference<ColorProfileName>(
    USER_PREFERENCES.FLAMEGRAPH_COLOR_PROFILE.key
  );

  return colorProfiles[colorProfile] ?? colorProfiles.ocean;
};

export default useCurrentColorProfile;
