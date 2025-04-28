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

export const DividerWithLabel = ({
  label,
  className = '',
  bgColorClassName = 'bg-gray-50 dark:bg-gray-900',
  textColorClassName = 'text-gray-700 dark:text-gray-300',
  dividerColorClassName = 'border-gray-300 dark:border-gray-700',
}: {
  label: string;
  className?: string;
    bgColorClassName?: string;
    textColorClassName?: string;
    dividerColorClassName?: string;
}): JSX.Element => {
  return (
    <div className={cx('relative', className)}>
      <div aria-hidden="true" className="absolute inset-0 flex items-center">
        <div className={`w-full border-t ${dividerColorClassName}`} />
      </div>
      <div className="relative flex justify-start">
        <span className={`${bgColorClassName} pr-2 text-xs ${textColorClassName} uppercase`}>{label}</span>
      </div>
    </div>
  );
};
