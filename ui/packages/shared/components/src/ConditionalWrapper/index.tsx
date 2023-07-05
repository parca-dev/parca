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

import {FC, PropsWithChildren, ReactNode} from 'react';

interface Props<T> {
  condition: boolean;
  children: ReactNode;
  WrapperComponent: FC<T>;
  wrapperProps: T;
}

export const ConditionalWrapper = <T extends PropsWithChildren>({
  condition,
  WrapperComponent,
  wrapperProps,
  children,
}: Props<T>): JSX.Element => {
  if (condition) {
    return <WrapperComponent {...wrapperProps}>{children}</WrapperComponent>;
  }

  return <>{children}</>;
};
