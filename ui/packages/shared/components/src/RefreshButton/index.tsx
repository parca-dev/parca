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

import {Icon} from '@iconify/react';
import cx from 'classnames';

interface RefreshButtonProps {
  onClick: () => void;
  disabled: boolean;
  title: string;
  testId: string;
  loading?: boolean;
  sticky?: boolean;
}

const RefreshButton = ({
  onClick,
  disabled,
  title,
  testId,
  loading,
  sticky = false,
}: RefreshButtonProps): JSX.Element => {
  return (
    <div
      className={cx(
        'w-full flex items-center justify-center px-3 py-2 bg-gray-50 dark:bg-gray-900 border-t border-gray-200 dark:border-gray-700',
        sticky && 'sticky bottom-0 z-20 mt-auto'
      )}
    >
      <button
        onClick={e => {
          e.preventDefault();
          e.stopPropagation();
          onClick();
        }}
        disabled={disabled}
        className={cx(
          'py-1 px-2 flex items-center gap-1 rounded-full transition-all duration-200 w-auto justify-center',
          disabled
            ? 'cursor-wait opacity-50'
            : 'hover:bg-gray-200 dark:hover:bg-gray-700 cursor-pointer'
        )}
        title={title}
        type="button"
        data-testid={testId}
      >
        <Icon
          icon="system-uicons:reset"
          className={cx('w-3 h-3 text-gray-500 dark:text-gray-400', disabled && 'animate-spin')}
        />
        <span className="text-xs text-gray-500 dark:text-gray-400">Refresh results</span>
      </button>
    </div>
  );
};

export default RefreshButton;
