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
import cx from 'classnames';

import {useUserPreference, type UserPreferenceDetails} from '@parca/hooks';

interface GenericInputProps {
  id: string;
  type: string;
  value: any;
  onChange: (value: any) => void;
  disabled?: boolean;
}

const GenericInput = ({
  id,
  type,
  value,
  onChange,
  disabled = false,
}: GenericInputProps): JSX.Element => {
  switch (type) {
    case 'boolean':
      return (
        <input
          className="mt-[5px] border bg-inherit disabled:cursor-not-allowed"
          id={id}
          type="checkbox"
          checked={value}
          onChange={e => onChange(e.target.checked)}
          disabled={disabled}
        />
      );
    case 'number':
      return (
        <input
          className="w-20 rounded-md border border-gray-200 bg-gray-50 p-1 px-2 disabled:cursor-not-allowed dark:border-gray-600 dark:bg-gray-900"
          id={id}
          type="number"
          min="0"
          value={value}
          onChange={e => onChange(Number(e.target.value))}
          disabled={disabled}
        />
      );
  }
  return <></>;
};

interface UserPreferenceItemProps {
  userPreferenceDetails: UserPreferenceDetails;
  className?: string;
  labelToLeft?: boolean;
  disabled?: boolean;
  inputSuffix?: string;
  id?: string;
}

// function UserPreferenceItem<T>({
//   userPreferenceDetails,
//   className = '',
//   labelToLeft = false,
//   disabled = false,
//   inputSuffix = '',
//   id = '',
// }: UserPreferenceItemProps): JSX.Element {
//   const [enabledTrimming, setEnabledTrimming] = useUserPreference<T>(userPreferenceDetails.key);

//   return (
//     <div
//       className={cx('mb-1 flex items-start gap-2', {
//         [className]: className,
//         'opacity-50': disabled,
//       })}
//     >
//       {labelToLeft ? (
//         <label htmlFor={userPreferenceDetails.key}>{userPreferenceDetails.name}</label>
//       ) : null}
//       <GenericInput
//         id={id ?? userPreferenceDetails.key}
//         value={enabledTrimming}
//         onChange={setEnabledTrimming}
//         type={userPreferenceDetails.type}
//         disabled={disabled}
//       />
//       {inputSuffix.length > 0 ? <span>{inputSuffix}</span> : null}
//       {!labelToLeft ? (
//         <label htmlFor={userPreferenceDetails.key}>{userPreferenceDetails.name}</label>
//       ) : null}
//     </div>
//   );
// }

function UserPreferenceItem<T>({
  userPreferenceDetails,
  className = '',
  disabled = false,
  id = '',
}: UserPreferenceItemProps): JSX.Element {
  const [enabledPreference, setEnabledPreference] = useUserPreference<T>(userPreferenceDetails.key);

  return (
    <div
      className={cx('flex items-start', {
        [className]: className,
        'opacity-50': disabled,
      })}
    >
      <div className="flex items-center justify-between">
        <div className="max-w-[625px]">
          <h2 className="text-base font-medium text-gray-800">{userPreferenceDetails.name}</h2>
          <p className="text-sm text-gray-700 mt-2">{userPreferenceDetails.description}</p>
        </div>
        <Switch
          id={id ?? userPreferenceDetails.key}
          checked={enabledPreference as boolean}
          onChange={(checked: boolean) => setEnabledPreference(checked as T)}
          className={`${(enabledPreference as boolean) ? 'bg-indigo-600' : 'bg-gray-400'}
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
    </div>
  );
}

export default UserPreferenceItem;
