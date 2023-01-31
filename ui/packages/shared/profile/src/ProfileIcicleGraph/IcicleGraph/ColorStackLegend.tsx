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
import {Icon} from '@iconify/react';
import useUserPreference, {USER_PREFERENCES} from '@parca/functions/useUserPreference';
import {EVERYTHING_ELSE, selectStackColors, useAppSelector} from '@parca/store';
import {useMemo} from 'react';
import {useURLState} from '@parca/components';
import {NavigateFunction} from '@parca/functions';

interface Props {
  navigateTo?: NavigateFunction;
}

const ColorStackLegend = ({navigateTo}: Props): JSX.Element => {
  const [colorProfileName] = useUserPreference<string>(
    USER_PREFERENCES.FLAMEGRAPH_COLOR_PROFILE.key
  );
  const [currentSearchString, setSearchString] = useURLState({param: 'search_string', navigateTo});
  const stackColors = useAppSelector(selectStackColors);

  const stackColorArray = useMemo(() => {
    return Object.entries(stackColors).sort(([featureA], [featureB]) => {
      if (featureA === EVERYTHING_ELSE) {
        return 1;
      }
      if (featureB === EVERYTHING_ELSE) {
        return -1;
      }
      return featureA?.localeCompare(featureB ?? '') ?? 0;
    });
  }, [stackColors]);

  if (colorProfileName === 'default') {
    return <></>;
  }
  return (
    <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-6 2xl:grid-cols-10 gap-2 px-10 my-6">
      {stackColorArray.map(([feature, color]) => {
        const filteringAllowed = feature !== EVERYTHING_ELSE;
        const isHighlighted = currentSearchString === feature;
        return (
          <div
            key={feature}
            className={cx('flex gap-1 items-center text-ellipsis p-1 justify-between pr-2', {
              'cursor-pointer': filteringAllowed,
              'bg-gray-200 dark:bg-gray-800': isHighlighted,
            })}
            onClick={() => {
              if (!filteringAllowed) {
                return;
              }
              if (isHighlighted) {
                setSearchString('');
                return;
              }
              setSearchString(feature);
            }}
          >
            <div className="flex items-center">
              <div className="w-4 h-4 mr-1 inline-block" style={{backgroundColor: color}} />
              <span className="text-sm text-ellipsis whitespace-nowrap overflow-hidden">
                {feature}
              </span>
            </div>
            {isHighlighted ? (
              <Icon
                icon="radix-icons:cross-circled"
                onClick={e => {
                  setSearchString('');
                  e.stopPropagation();
                }}
              />
            ) : null}
          </div>
        );
      })}
    </div>
  );
};

export default ColorStackLegend;
