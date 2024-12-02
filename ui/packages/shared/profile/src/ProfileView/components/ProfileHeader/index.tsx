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

import {FC} from 'react';

import cx from 'classnames';

interface ProfileHeaderProps {
  profileSourceString?: string;
  hasProfileSource: boolean;
  externalMainActions?: React.ReactNode;
}

export const ProfileHeader: FC<ProfileHeaderProps> = ({
  profileSourceString,
  hasProfileSource,

  externalMainActions,
}) => {
  const headerParts = profileSourceString?.split('"') ?? [];

  const showDivider =
    hasProfileSource && (externalMainActions === null || externalMainActions === undefined);

  return (
    <>
      {showDivider && (
        <div className="border-t border-gray-200 dark:border-gray-700 h-[1px] w-full pb-4" />
      )}
      <div
        className={cx(
          'flex w-full',
          hasProfileSource || externalMainActions != null ? 'justify-start' : 'justify-end',
          {
            'items-end mb-4': !hasProfileSource && externalMainActions != null,
            'items-center mb-2': hasProfileSource,
          }
        )}
      >
        <div>
          {hasProfileSource && (
            <div className="flex items-center gap-1">
              <div className="text-xs font-medium">
                {headerParts.length > 0 ? headerParts[0].replace(/"/g, '') : ''}
              </div>
              <div className="text-xs font-medium">
                {headerParts.length > 1
                  ? headerParts[headerParts.length - 1].replace(/"/g, '')
                  : ''}
              </div>
            </div>
          )}
          {externalMainActions}
        </div>
      </div>
    </>
  );
};
