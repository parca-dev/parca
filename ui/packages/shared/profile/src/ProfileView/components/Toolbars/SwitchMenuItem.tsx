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

import {Switch} from '@headlessui/react';

import {useUserPreference, type UserPreferenceDetails} from '@parca/hooks';

interface SwitchMenuItemProps {
  label: string;
  id: string;
  userPreferenceDetails: UserPreferenceDetails;
}

function SwitchMenuItem<T>({label, id, userPreferenceDetails}: SwitchMenuItemProps): JSX.Element {
  const [enabledPreference, setEnabledPreference] = useUserPreference<T>(userPreferenceDetails.key);

  return (
    <div className="flex items-center justify-between w-full">
      <span>{label}</span>
      <Switch
        id={id}
        checked={enabledPreference as boolean}
        onChange={(checked: boolean) => setEnabledPreference(checked as T)}
        className={`${
          (enabledPreference as boolean) ? 'bg-indigo-600' : 'bg-gray-400 dark:bg-gray-800'
        }
        relative inline-flex h-[24px] w-[44px] shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus-visible:ring-2  focus-visible:ring-white/75`}
      >
        <span className="sr-only">Use setting</span>
        <span
          aria-hidden="true"
          className={`${(enabledPreference as boolean) ? 'translate-x-5' : 'translate-x-0'}
            pointer-events-none inline-block h-[20px] w-[20px] transform rounded-full bg-white shadow-lg ring-0 transition duration-200 ease-in-out`}
        />
      </Switch>
    </div>
  );
}

export default SwitchMenuItem;
