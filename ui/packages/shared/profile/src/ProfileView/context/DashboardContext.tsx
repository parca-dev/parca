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

import {FC, PropsWithChildren, createContext, useContext} from 'react';

import {useURLState} from '@parca/components';

import {VisualizationType} from '../types/visualization';

interface DashboardContextType {
  dashboardItems: string[];
  setDashboardItems: (items: string[]) => void;
  handleClosePanel: (visualizationType: VisualizationType) => void;
  isMultiPanelView: boolean;
}

const DashboardContext = createContext<DashboardContextType | undefined>(undefined);

export const DashboardProvider: FC<PropsWithChildren> = ({children}) => {
  const [dashboardItems, setDashboardItems] = useURLState<string[]>('dashboard_items', {
    alwaysReturnArray: true,
  });

  const handleClosePanel = (visualizationType: VisualizationType): void => {
    const newDashboardItems = dashboardItems.filter(item => item !== visualizationType);
    setDashboardItems(newDashboardItems);
  };

  const isMultiPanelView = dashboardItems.length > 1;

  return (
    <DashboardContext.Provider
      value={{
        dashboardItems,
        setDashboardItems,
        handleClosePanel,
        isMultiPanelView,
      }}
    >
      {children}
    </DashboardContext.Provider>
  );
};

export const useDashboard = (): DashboardContextType => {
  const context = useContext(DashboardContext);
  if (context === undefined) {
    throw new Error('useDashboard must be used within a DashboardProvider');
  }
  return context;
};
