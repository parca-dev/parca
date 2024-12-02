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
import {
  DragDropContext,
  Draggable,
  Droppable,
  type DraggableLocation,
  type DropResult,
} from 'react-beautiful-dnd';

import {useDashboard} from '../../context/DashboardContext';
import {VisualizationType} from '../../types/visualization';
import {VisualizationContainer} from '../VisualizationContainer';

interface DashboardLayoutProps {
  getDashboardItemByType: (props: {type: VisualizationType; isHalfScreen: boolean}) => JSX.Element;
  actionButtons: {
    icicle: JSX.Element;
    table: JSX.Element;
  };
}

export const DashboardLayout: FC<DashboardLayoutProps> = ({
  getDashboardItemByType,
  actionButtons,
}) => {
  const {dashboardItems, setDashboardItems, isMultiPanelView} = useDashboard();

  const onDragEnd = (result: DropResult): void => {
    const {destination, source, draggableId} = result;

    if (Boolean(destination) && destination?.index !== source.index) {
      const targetItem = draggableId;
      const otherItems = dashboardItems.filter(item => item !== targetItem);
      const newDashboardItems =
        (destination as DraggableLocation).index < source.index
          ? [targetItem, ...otherItems]
          : [...otherItems, targetItem];

      setDashboardItems(newDashboardItems);
    }
  };

  return (
    <DragDropContext onDragEnd={onDragEnd}>
      <Droppable droppableId="droppable" direction="horizontal">
        {provided => (
          <div
            ref={provided.innerRef}
            className={cx(
              'grid w-full gap-2',
              isMultiPanelView ? 'grid-cols-2 mt-4' : 'grid-cols-1'
            )}
            {...provided.droppableProps}
          >
            {dashboardItems.map((dashboardItem, index) => (
              <Draggable
                key={dashboardItem}
                draggableId={dashboardItem}
                index={index}
                isDragDisabled={!isMultiPanelView}
              >
                {(provided, snapshot) => (
                  <VisualizationContainer
                    provided={provided}
                    snapshot={snapshot}
                    dashboardItem={dashboardItem as VisualizationType}
                    getDashboardItemByType={getDashboardItemByType}
                    isMultiPanelView={isMultiPanelView}
                    index={index}
                    actionButtons={actionButtons}
                  />
                )}
              </Draggable>
            ))}
            {provided.placeholder}
          </div>
        )}
      </Droppable>
    </DragDropContext>
  );
};
