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

import {forwardRef, useImperativeHandle, useState} from 'react';

import {Table} from '@uwdata/flechette';

import {ProfileType} from '@parca/parser';

import ContextMenu from './ContextMenu';

interface ContextMenuWrapperProps {
  menuId: string;
  table: Table;
  total: bigint;
  totalUnfiltered: bigint;
  profileType?: ProfileType;
  compareAbsolute: boolean;
  resetPath: () => void;
  hideMenu: () => void;
  hideBinary: (binaryToRemove: string) => void;
  unit?: string;
  isInSandwichView?: boolean;
}

export interface ContextMenuWrapperRef {
  setRow: (row: number, callback?: () => void) => void;
}

const ContextMenuWrapper = forwardRef<ContextMenuWrapperRef, ContextMenuWrapperProps>(
  (props, ref) => {
    // Initialize with null to prevent rendering with invalid data
    const [row, setRow] = useState<number | null>(null);

    useImperativeHandle(ref, () => ({
      setRow: (newRow: number, callback?: () => void) => {
        setRow(newRow);
        // Execute callback after state update using requestAnimationFrame
        if (callback != null) {
          requestAnimationFrame(() => {
            requestAnimationFrame(callback);
          });
        }
      },
    }));

    // Only render ContextMenu when we have a valid row
    if (row === null) {
      return null;
    }

    return <ContextMenu {...props} row={row} isSandwich={props.isInSandwichView} />;
  }
);

ContextMenuWrapper.displayName = 'ContextMenuWrapper';

export default ContextMenuWrapper;
