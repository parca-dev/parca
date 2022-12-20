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

import useUserPreference, {USER_PREFERENCES} from '@parca/functions/useUserPreference';
import UserPreferenceItem from './UserPreferenceItem';

const GraphTrimming = () => {
  const [disabledTrimming] = useUserPreference<boolean>(USER_PREFERENCES.DISABLE_TRIMMING.key);

  return (
    <div>
      <UserPreferenceItem userPreferenceDetails={USER_PREFERENCES.DISABLE_TRIMMING} />
      <div className="">
        <UserPreferenceItem
          userPreferenceDetails={USER_PREFERENCES.NODE_TRIM_THRESHOLD}
          className="flex-row-reverse w-80"
          disabled={disabledTrimming}
        />
        <p className="text-xs mt-1 ml-6">
          Nodes that have value less than this percentage will be removed.
        </p>
      </div>
    </div>
  );
};

export default GraphTrimming;
