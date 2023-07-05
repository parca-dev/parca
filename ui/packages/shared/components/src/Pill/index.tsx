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

const VARIANTS = {
  primary: {
    color: 'text-gray-600 dark:text-gray-200',
    bg: 'bg-indigo-100 dark:bg-indigo-900',
  },
  success: {
    color: 'text-green-800 dark:text-green-200',
    bg: 'bg-green-100 dark:bg-green-900',
  },
  danger: {
    color: 'text-red-800 dark:text-red-200',
    bg: 'bg-red-100 dark:bg-red-900',
  },
  warning: {
    color: 'text-amber-800 dark:text-amber-200',
    bg: 'bg-amber-100 dark:bg-amber-900',
  },
  info: {
    color: 'text-blue-600 dark:text-blue-200',
    bg: 'bg-blue-100 dark:bg-blue-800',
  },
  neutral: {
    color: 'text-neutral-800 dark:text-neutral-200',
    bg: 'bg-neutral-100 dark:bg-neutral-900',
  },
};

export type PillVariant = keyof typeof VARIANTS;

const Pill = ({
  variant = 'primary',
  children,
  ...props
}: {
  variant?: PillVariant;
  children: React.ReactNode;
} & JSX.IntrinsicElements['span']): JSX.Element => (
  <span
    className={cx(
      VARIANTS[variant].color,
      VARIANTS[variant].bg,
      `inline-flex whitespace-nowrap rounded-full px-2 text-xs font-semibold leading-5 `
    )}
    {...props}
  >
    {children}
  </span>
);

export default Pill;
