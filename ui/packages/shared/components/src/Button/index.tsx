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

import {useMemo} from 'react';

import {Icon} from '@iconify/react';
import cx from 'classnames';
import {twMerge} from 'tailwind-merge';

const BUTTON_VARIANT_CONFIG = {
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
} as const;

export type ButtonVariant = keyof typeof BUTTON_VARIANT_CONFIG;

type Props = {
  disabled?: boolean;
  /**
   * variant specifies the variant of the button to be rendered
   */
  variant?: ButtonVariant;
  className?: string;
  children: React.ReactNode;
} & JSX.IntrinsicElements['button'];

export const Button = ({
  variant = 'primary',
  disabled = false,
  className = '',
  ...props
}: Props): JSX.Element => {
  const classes = useMemo<string>(() => {
    const variantConfig = BUTTON_VARIANT_CONFIG[variant];

    const classes = cx(
      'flex text-sm rounded-md focus:outline-none focus:ring-2 focus:ring-offset-2 items-center relative',
      ...Object.values(variantConfig),
      {'opacity-50 pointer-events-none': disabled},
      {[className]: className}
    );
    const classesMerged = twMerge(classes);

    if (classes.length > classesMerged.length) {
      const classesTokens = classes.split(' ');
      const classesMergedTokens = classesMerged.split(' ');
      console.warn(
        'Button: Conflicting classes found in `className` prop, please use/create an appropriate variant instead. Conflicting classes:',
        classesTokens.filter(token => !classesMergedTokens.includes(token))
      );
    }

    return classesMerged;
  }, [className, disabled, variant]);

  return <button {...props} disabled={disabled} className={classes} />;
};

type IconButtonProps = {
  /** Icon to display in the button.
   * Can be a string (icon name from @iconify/react) or a ReactNode (e.g. an SVG)
   */
  icon: string | React.ReactNode;
} & Omit<Props, 'variant' | 'children'>;

export const IconButton = ({icon, ...props}: IconButtonProps): JSX.Element => {
  return (
    <Button {...props} variant="link">
      {typeof icon === 'string' ? <Icon icon={icon} /> : icon}
    </Button>
  );
};
