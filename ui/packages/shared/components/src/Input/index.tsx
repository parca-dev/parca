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
  actionButton?: JSX.Element;
  id?: string;
}

export type Props = React.InputHTMLAttributes<HTMLInputElement> & SelfProps;

const Input = ({
  className = '',
  onAction,
  actionIcon = <Icon icon="ep:arrow-right" />,
  actionButton,
  onBlur,
  id = '',
  ...props
}: Props): JSX.Element => {
  const ref = useRef<HTMLInputElement>(null);
  const hasAction = onAction != null;
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
          'relative rounded-md border border-gray-200 bg-white p-2 dark:border-gray-600 dark:bg-gray-900 focus:outline-none focus:ring-1 focus:z-2',
          {
            [className]: className.length > 0,
            '!pr-8': hasAction,
          }
        )}
        onKeyDown={e => {
          if (e.key === 'Enter' && hasAction) {
            onAction();
          }
        }}
        id={id}
      />
      {hasAction ? (
        <div
          className={cx(
            '!absolute inset-y-[1px] right-[1px] rounded-bl-none rounded-tl-none rounded-md w-8 z-4'
          )}
        >
          {actionButton != null ? (
            actionButton
          ) : (
            <Button variant="secondary" onClick={onAction} className={cx('h-full !px-2')}>
              {actionIcon}
            </Button>
          )}
        </div>
      ) : null}
    </div>
  );
};

export default Input;
