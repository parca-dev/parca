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

import React, {useState} from 'react';

import {Icon} from '@iconify/react';
import cx from 'classnames';
import type {DraggableProvidedDragHandleProps} from 'react-beautiful-dnd';

import {IconButton} from '@parca/components';
import {CloseIcon} from '@parca/icons';
import type {NavigateFunction} from '@parca/utilities';

import ViewSelector from './ViewSelector';

interface Props {
  dashboardItem: string;
  index: number;
  isMultiPanelView: boolean;
  handleClosePanel: (dashboardItem: string) => void;
  navigateTo: NavigateFunction | undefined;
  dragHandleProps: DraggableProvidedDragHandleProps | null | undefined;
  getDashboardItemByType: (props: {
    type: string;
    isHalfScreen: boolean;
    setActionButtons: (actionButtons: JSX.Element) => void;
  }) => JSX.Element;
}

export const VisualizationPanel = React.memo(function VisualizationPanel({
  dashboardItem,
  index,
  isMultiPanelView,
  handleClosePanel,
  navigateTo,
  dragHandleProps,
  getDashboardItemByType,
}: Props): JSX.Element {
  const [actionButtons, setActionButtons] = useState<JSX.Element>(<></>);

  return (
    <>
      <div className="flex w-full justify-end gap-2 pb-2">
        <div className="flex w-full items-center justify-between">
          <div className="flex">
            <div
              className={cx(isMultiPanelView ? 'visible' : 'invisible', 'flex items-center')}
              {...dragHandleProps}
            >
              <Icon className="text-xl" icon="material-symbols:drag-indicator" />
            </div>
            <>{actionButtons}</>
          </div>
          <ViewSelector defaultValue={dashboardItem} navigateTo={navigateTo} position={index} />
        </div>

        {isMultiPanelView && (
          <IconButton onClick={() => handleClosePanel(dashboardItem)} icon={<CloseIcon />} />
        )}
      </div>
      {getDashboardItemByType({
        type: dashboardItem,
        isHalfScreen: isMultiPanelView,
        setActionButtons,
      })}
    </>
  );
});
