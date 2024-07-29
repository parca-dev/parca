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

import {USER_PREFERENCES} from '@parca/hooks';

import FlamegraphColorProfileSelector from './FlamegraphColorProfileSelector';
import UserPreferenceItem from './UserPreferenceItem';

const UserPreferences = (): JSX.Element => {
  return (
    <div>
      <div className="min-h-40 mt-8">
        <UserPreferenceItem
          id="h-highlight-after-filtering"
          userPreferenceDetails={USER_PREFERENCES.HIGHTLIGHT_AFTER_FILTERING}
        />
        <UserPreferenceItem
          id="h-dock-graph-tooltip"
          userPreferenceDetails={USER_PREFERENCES.GRAPH_METAINFO_DOCKED}
        />
        <UserPreferenceItem
          id="h-highlight-similar-stacks"
          userPreferenceDetails={USER_PREFERENCES.HIGHLIGHT_SIMILAR_STACKS}
        />
        <FlamegraphColorProfileSelector />
      </div>
    </div>
  );
};

export default UserPreferences;
