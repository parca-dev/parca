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

import React from 'react';

import cx from 'classnames';

export interface CardProps {
  children: React.ReactNode;
  className?: string;
}

const Card = ({children, className = ''}: CardProps): JSX.Element => {
  return (
    <div className="mx-auto">
      <div
        className={cx(
          'g-gray flex-column flex-1 overflow-hidden bg-white shadow dark:bg-gray-700 sm:rounded-lg',
          {[className]: className.length}
        )}
      >
        {children}
      </div>
    </div>
  );
};

const Header = ({
  children,
  className = '',
}: {
  children: React.ReactNode;
  className?: string;
}): JSX.Element => {
  return (
    <div
      className={cx('bg-gray-200 px-4 py-4 dark:bg-gray-800 sm:px-6', {
        [className]: className.length,
      })}
      style={{justifyContent: 'space-between', alignItems: 'stretch'}}
    >
      {children}
    </div>
  );
};

const Body = ({children}: {children: React.ReactNode}): JSX.Element => {
  return <div className="p-4">{children}</div>;
};

export default Object.assign(Card, {
  Header,
  Body,
});
