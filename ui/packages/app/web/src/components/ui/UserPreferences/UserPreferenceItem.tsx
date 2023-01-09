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

import cx from 'classnames';
import useUserPreference, {UserPreferenceDetails} from '@parca/functions/useUserPreference';

interface GenericInputProps {
  id: string;
  type: string;
  value: any;
  onChange: (value: any) => void;
  disabled?: boolean;
}

const GenericInput = ({id, type, value, onChange, disabled = false}: GenericInputProps) => {
  switch (type) {
    case 'boolean':
      return (
        <input
          className="bg-inherit border disabled:cursor-not-allowed"
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
          className="bg-gray-50 dark:bg-gray-900 border border-gray-200 dark:border-gray-600 p-1 px-2 w-20 disabled:cursor-not-allowed rounded-md"
          id={id}
          type="number"
          min="0"
          value={value}
          onChange={e => onChange(Number(e.target.value))}
          disabled={disabled}
        />
      );
  }
  return null;
};

interface UserPreferenceItemProps {
  userPreferenceDetails: UserPreferenceDetails;
  className?: string;
  labelToLeft?: boolean;
  disabled?: boolean;
  inputSuffix?: string;
}

function UserPreferenceItem<T>({
  userPreferenceDetails,
  className = '',
  labelToLeft = false,
  disabled = false,
  inputSuffix = '',
}: UserPreferenceItemProps) {
  const [enabledTrimming, setEnabledTrimming] = useUserPreference<T>(userPreferenceDetails.key);

  return (
    <div
      className={cx('flex gap-2 items-center', {[className]: className, 'opacity-50': disabled})}
    >
      {labelToLeft ? (
        <label htmlFor={userPreferenceDetails.key}>{userPreferenceDetails.name}</label>
      ) : null}
      <GenericInput
        id={userPreferenceDetails.key}
        value={enabledTrimming}
        onChange={setEnabledTrimming}
        type={userPreferenceDetails.type}
        disabled={disabled}
      />
      {inputSuffix ? <span>{inputSuffix}</span> : null}
      {!labelToLeft ? (
        <label htmlFor={userPreferenceDetails.key}>{userPreferenceDetails.name}</label>
      ) : null}
    </div>
  );
}

export default UserPreferenceItem;
