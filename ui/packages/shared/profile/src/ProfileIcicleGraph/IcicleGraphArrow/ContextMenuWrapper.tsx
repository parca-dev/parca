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

import {forwardRef, useEffect, useImperativeHandle, useRef, useState} from 'react';

import {Table} from 'apache-arrow';
import {flushSync} from 'react-dom';

import {ProfileType} from '@parca/parser';

import ContextMenu from './ContextMenu';

interface ContextMenuWrapperProps {
  menuId: string;
  table: Table<any>;
  total: bigint;
  totalUnfiltered: bigint;
  profileType?: ProfileType;
  compareAbsolute: boolean;
  resetPath: () => void;
  hideMenu: () => void;
  hideBinary: (binaryToRemove: string) => void;
  unit?: string;
  isSandwich?: boolean;
}

export interface ContextMenuWrapperRef {
  setRow: (row: number, callback?: () => void) => void;
}

const ContextMenuWrapper = forwardRef<ContextMenuWrapperRef, ContextMenuWrapperProps>(
  (props, ref) => {
    // Fix for race condition: Start with null instead of 0 to prevent initial render
    // with invalid row data. This ensures ContextMenu only renders when we have
    // the correct row number from the right-click event.
    const [row, setRow] = useState<number | null>(null);
    const callbackRef = useRef<(() => void) | null>(null);

    // Execute callback after row state update completes
    useEffect(() => {
      if (callbackRef.current && row !== null) {
        callbackRef.current();
        callbackRef.current = null;
      }
    }, [row]);

    useImperativeHandle(ref, () => ({
      setRow: (newRow: number, callback?: () => void) => {
        if (callback) {
          callbackRef.current = callback;
        }
        // Use flushSync to ensure synchronous state update before callback execution
        flushSync(() => {
          setRow(newRow);
        });
      },
    }));

    // Don't render the ContextMenu until we have a valid row.
    if (row === null) {
      return null;
    }

    return <ContextMenu {...props} row={row} />;
  }
);

ContextMenuWrapper.displayName = 'ContextMenuWrapper';

export default ContextMenuWrapper;
