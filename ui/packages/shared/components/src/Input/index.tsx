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

const Input = ({className = '', ...props}) => {
  return (
    <input
      {...props}
      className={cx(
        'p-2 rounded-md bg-gray-50 dark:bg-gray-900 border border-gray-200 dark:border-gray-600',
        {
          [className]: className.length > 0,
        }
      )}
    />
  );
};

export default Input;
