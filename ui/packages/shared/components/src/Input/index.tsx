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

import {useRef} from 'react';

import {Icon} from '@iconify/react';
import cx from 'classnames';

import {Button} from '../Button';

interface SelfProps {
  className?: string;
  onAction?: () => void;
  actionIcon?: JSX.Element;
}

export type Props = React.InputHTMLAttributes<HTMLInputElement> & SelfProps;

const Input = ({
  className = '',
  onAction,
  actionIcon = <Icon icon="ep:arrow-right" />,
  onBlur,
  ...props
}: Props): JSX.Element => {
  const ref = useRef<HTMLInputElement>(null);
  return (
    <div
      className="relative"
      ref={ref}
      onBlur={e => {
        (async () => {
          if (onBlur == null || ref.current == null) {
            return;
          }
          await new Promise(resolve => setTimeout(resolve));
          if (ref.current.contains(document.activeElement)) {
            return;
          }
          onBlur(e as React.FocusEvent<HTMLInputElement>);
        })().catch(err => {
          console.error('Error in processing blur event', err);
        });
      }}
    >
      <input
        {...props}
        className={cx(
          'rounded-md border border-gray-200 bg-gray-50 p-2 dark:border-gray-600 dark:bg-gray-900',
          {
            [className]: className.length > 0,
            'pr-10': onAction != null,
          }
        )}
        onKeyDown={e => {
          if (e.key === 'Enter' && onAction != null) {
            onAction();
          }
        }}
      />
      {onAction != null ? (
        <Button
          onClick={onAction}
          className="!absolute inset-y-0 right-0 !w-auto rounded-tl-none rounded-bl-none !px-2"
        >
          {actionIcon}
        </Button>
      ) : null}
    </div>
  );
};

export default Input;
