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

import React, {createContext, useCallback, useContext, useMemo, useRef} from 'react';

import {Table} from 'apache-arrow';

import {ProfileType} from '@parca/parser';

interface TooltipState {
  row: number | null;
  x: number;
  y: number;
}

interface TooltipContextValue {
  table: Table<any>;
  total: bigint;
  totalUnfiltered: bigint;
  profileType?: ProfileType;
  unit?: string;
  compareAbsolute: boolean;
  updateTooltip: (row: number | null, x?: number, y?: number) => void;
  tooltipState: TooltipState;
}

const TooltipContext = createContext<TooltipContextValue | null>(null);

export const useTooltipContext = (): TooltipContextValue => {
  const context = useContext(TooltipContext);
  if (context === undefined || context === null) {
    throw new Error('useTooltipContext must be used within TooltipProvider');
  }
  return context;
};

interface TooltipProviderProps {
  children: React.ReactNode;
  table: Table<any>;
  total: bigint;
  totalUnfiltered: bigint;
  profileType?: ProfileType;
  unit?: string;
  compareAbsolute: boolean;
  onTooltipUpdate?: (state: TooltipState) => void;
}

export const TooltipProvider: React.FC<TooltipProviderProps> = ({
  children,
  table,
  total,
  totalUnfiltered,
  profileType,
  unit,
  compareAbsolute,
  onTooltipUpdate,
}) => {
  const tooltipStateRef = useRef<TooltipState>({row: null, x: 0, y: 0});

  const updateTooltip = useCallback(
    (row: number | null, x = 0, y = 0) => {
      tooltipStateRef.current = {row, x, y};
      onTooltipUpdate?.(tooltipStateRef.current);
    },
    [onTooltipUpdate]
  );

  const value = useMemo(
    () => ({
      table,
      total,
      totalUnfiltered,
      profileType,
      unit,
      compareAbsolute,
      updateTooltip,
      tooltipState: tooltipStateRef.current,
    }),
    [table, total, totalUnfiltered, profileType, unit, compareAbsolute, updateTooltip]
  );

  return <TooltipContext.Provider value={value}>{children}</TooltipContext.Provider>;
};
