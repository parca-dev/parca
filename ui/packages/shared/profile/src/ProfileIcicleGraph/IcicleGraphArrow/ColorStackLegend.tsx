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

import React, {useMemo} from 'react';

import {Icon} from '@iconify/react';
import cx from 'classnames';

import {useURLState} from '@parca/components';
import {USER_PREFERENCES, useUserPreference} from '@parca/hooks';
import {EVERYTHING_ELSE} from '@parca/store';
import type {NavigateFunction} from '@parca/utilities';

import {mappingColors} from './IcicleGraphNodes';

interface Props {
  mappingColors: mappingColors;
  navigateTo?: NavigateFunction;
  compareMode?: boolean;
}

const ColorStackLegend = ({
  mappingColors,
  navigateTo,
  compareMode = false,
}: Props): React.JSX.Element => {
  const [colorProfileName] = useUserPreference<string>(
    USER_PREFERENCES.FLAMEGRAPH_COLOR_PROFILE.key
  );
  const [currentSearchString, setSearchString] = useURLState({param: 'search_string', navigateTo});

  const stackColorArray = useMemo(() => {
    return Object.entries(mappingColors).sort(([featureA], [featureB]) => {
      if (featureA === EVERYTHING_ELSE) {
        return 1;
      }
      if (featureB === EVERYTHING_ELSE) {
        return -1;
      }
      return featureA?.localeCompare(featureB ?? '') ?? 0;
    });
  }, [mappingColors]);

  if (mappingColors === undefined) {
    return <></>;
  }

  if (Object.entries(mappingColors).length === 0) {
    return <></>;
  }

  if (colorProfileName === 'default' || compareMode) {
    return <></>;
  }

  return (
    <div className="my-6 flex flex-wrap justify-center gap-2">
      {stackColorArray.map(([feature, color]) => {
        const filteringAllowed = feature !== EVERYTHING_ELSE;
        const isHighlighted = currentSearchString === feature;
        return (
          <div
            key={feature}
            className={cx('flex items-center justify-between gap-1 text-ellipsis p-1 pr-2', {
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
              <div className="mr-1 inline-block h-4 w-4" style={{backgroundColor: color}} />
              <span className="text-sm">{feature}</span>
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
