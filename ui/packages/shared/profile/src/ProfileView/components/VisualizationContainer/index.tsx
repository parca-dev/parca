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

import {FC} from 'react';

import cx from 'classnames';
import type {DraggableProvided, DraggableStateSnapshot} from 'react-beautiful-dnd';

import {useDashboard} from '../../context/DashboardContext';
import {VisualizationType} from '../../types/visualization';
import {VisualizationPanel} from '../VisualizationPanel';

interface VisualizationContainerProps {
  provided: DraggableProvided;
  snapshot: DraggableStateSnapshot;
  dashboardItem: VisualizationType;
  getDashboardItemByType: (props: {type: VisualizationType; isHalfScreen: boolean}) => JSX.Element;
  isMultiPanelView: boolean;
  index: number;
  actionButtons: {
    flame: JSX.Element;
    table: JSX.Element;
  };
}

export const VisualizationContainer: FC<VisualizationContainerProps> = ({
  provided,
  snapshot,
  dashboardItem,
  getDashboardItemByType,
  isMultiPanelView,
  index,
  actionButtons,
}) => {
  const {handleClosePanel} = useDashboard();

  return (
    <div
      ref={provided.innerRef}
      {...provided.draggableProps}
      className={cx(
        'w-full min-h-96',
        snapshot.isDragging ? 'bg-gray-200 dark:bg-gray-500' : 'bg-inherit dark:bg-gray-900',
        isMultiPanelView ? 'border-2 border-gray-100 dark:border-gray-700 rounded-md p-3' : '',
        dashboardItem === 'source' && isMultiPanelView ? 'sticky top-0 self-start' : ''
      )}
      style={
        dashboardItem === 'source' && isMultiPanelView
          ? {maxHeight: 'calc(100vh - 50px)'}
          : undefined
      }
    >
      <VisualizationPanel
        handleClosePanel={handleClosePanel}
        isMultiPanelView={isMultiPanelView}
        dashboardItem={dashboardItem}
        getDashboardItemByType={getDashboardItemByType}
        dragHandleProps={provided.dragHandleProps}
        index={index}
        actionButtons={actionButtons}
      />
    </div>
  );
};
