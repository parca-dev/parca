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

import {useEffect, useRef} from 'react';

interface Props {
  value: number;
  isCurrent?: boolean;
}

export const LineNo = ({value, isCurrent = false}: Props): JSX.Element => {
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (isCurrent) {
      ref.current?.scrollIntoView({behavior: 'smooth', block: 'center'});
    }
  }, [isCurrent]);

  return <code ref={ref}>{value.toString() + '\n'}</code>;
};
