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

import {createContext, ReactNode, useContext, useEffect, useMemo, useState} from 'react';

export interface KeyDownState {
  isShiftDown: boolean;
}

const DEFAULT_VALUE = {
  isShiftDown: false,
};

const KeyDownContext = createContext<KeyDownState>(DEFAULT_VALUE);

export const KeyDownProvider = ({
  children,
}: {
  children: ReactNode;
  value?: KeyDownState;
}): JSX.Element => {
  const [isShiftDown, setIsShiftDown] = useState<boolean>(false);

  useEffect(() => {
    const handleShiftDown = (event: {keyCode: number}): void => {
      if (event.keyCode === 16) {
        setIsShiftDown(true);
      }
    };

    window.addEventListener('keydown', handleShiftDown);

    const handleShiftUp = (event: {keyCode: number}): void => {
      if (event.keyCode === 16) {
        setIsShiftDown(false);
      }
    };

    window.addEventListener('keyup', handleShiftUp);

    return () => {
      window.removeEventListener('keydown', handleShiftDown);
      window.removeEventListener('keyup', handleShiftUp);
    };
  }, []);

  const value = useMemo(() => ({isShiftDown}), [isShiftDown]);

  return <KeyDownContext.Provider value={value}>{children}</KeyDownContext.Provider>;
};

export const useKeyDown = (): KeyDownState => {
  const context = useContext(KeyDownContext);
  if (context == null) {
    return DEFAULT_VALUE;
  }
  return context;
};

export default KeyDownContext;
