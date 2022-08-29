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
    color: 'text-gray-600',
    bg: 'bg-indigo-100',
  },
  success: {
    color: 'text-green-800',
    bg: 'bg-green-100',
  },
  danger: {
    color: 'text-red-800',
    bg: 'bg-red-100',
  },
  warning: {
    color: 'text-amber-800',
    bg: 'bg-amber-100',
  },
  info: {
    color: 'text-blue-600',
    bg: 'bg-blue-100',
  },
  neutral: {
    color: 'text-neutral-800',
    bg: 'bg-neutral-100',
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
      `px-2 inline-flex text-xs leading-5 font-semibold rounded-full whitespace-nowrap `
    )}
    {...props}
  >
    {children}
  </span>
);

export default Pill;
