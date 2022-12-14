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

const BUTTON_VARIANT = {
  primary: {
    text: 'text-gray-100 dark-gray-900 justify-center',
    bg: 'bg-indigo-600',
    border: 'border border-indigo-500',
    fontWeight: 'font-medium',
    hover: '',
    padding: 'py-2 px-4',
  },
  neutral: {
    text: 'text-gray-600 dark:text-gray-100 justify-center',
    bg: 'bg-gray-50 dark:bg-gray-900',
    border: 'border border-gray-200 dark:border-gray-600',
    fontWeight: 'font-normal',
    hover: '',
    padding: 'py-2 px-4',
  },
  link: {
    text: 'text-gray-600 dark:text-gray-300 justify-start',
    bg: '',
    border: '',
    fontWeight: 'font-normal',
    hover: 'hover:underline p-0',
    padding: 'py-1',
  },
};

export type ButtonVariant = keyof typeof BUTTON_VARIANT;

type Props = {
  disabled?: boolean;
  variant?: ButtonVariant;
  className?: string;
  children: React.ReactNode;
} & JSX.IntrinsicElements['button'];

const Button = ({
  disabled = false,
  variant = 'primary',
  children,
  className = '',
  ...props
}: Props): JSX.Element => {
  return (
    <button
      type="button"
      className={cx(
        disabled ? 'opacity-50 pointer-events-none' : '',
        ...Object.values(BUTTON_VARIANT[variant]),
        'cursor-pointer group relative w-full flex text-sm rounded-md text-whitefocus:outline-none focus:ring-2 focus:ring-offset-2 items-center justify-center',
        className
      )}
      disabled={disabled}
      {...props}
    >
      {children}
    </button>
  );
};

export default Button;

export const IconButton = ({className = '', ...props}: Exclude<Props, 'variant'>): JSX.Element => {
  return <Button {...props} variant="link" className={`w-fit ${className}`} />;
};

export const PageButton = ({children, className = '', ...rest}: Props): JSX.Element => {
  return (
    <button
      type="button"
      className={cx(
        'relative inline-flex items-center px-2 py-2 border border-gray-300 dark:border-none bg-white dark:bg-gray-900 text-sm font-medium text-gray-500 hover:bg-gray-50',
        className
      )}
      {...rest}
    >
      {children}
    </button>
  );
};
