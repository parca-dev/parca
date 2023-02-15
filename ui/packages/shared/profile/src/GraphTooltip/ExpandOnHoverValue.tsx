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

import {useEffect, useState} from 'react';

interface Props {
  value: string | number | undefined;
  displayValue?: string | number | undefined;
}

export const ExpandOnHover = ({value, displayValue}: Props): JSX.Element => {
  return (
    <div className="relative group w-full">
      <div className="text-ellipsis w-full overflow-hidden whitespace-nowrap">
        {displayValue ?? value}
      </div>
      <div className="group-hover:flex hidden absolute -inset-2 max-w-[500px] whitespace-normal h-fit bg-gray-50 dark:bg-gray-900 shadow-[0_0_10px_2px_rgba(0,0,0,0.3)] rounded p-2 break-all">
        {value}
      </div>
    </div>
  );
};
