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

import React from 'react';

import {Icon} from '@iconify/react';
import cx from 'classnames';
import type {DraggableProvidedDragHandleProps} from 'react-beautiful-dnd';

import {IconButton, useParcaContext} from '@parca/components';
import {CloseIcon} from '@parca/icons';

import {VisualizationType} from '../types/visualization';

interface Props {
  dashboardItem: VisualizationType;
  index: number;
  isMultiPanelView: boolean;
  handleClosePanel: (dashboardItem: VisualizationType) => void;
  dragHandleProps: DraggableProvidedDragHandleProps | null | undefined;
  getDashboardItemByType: (props: {type: VisualizationType; isHalfScreen: boolean}) => JSX.Element;
  actionButtons: {
    icicle: JSX.Element;
    table: JSX.Element;
  };
}

export const VisualizationPanel = React.memo(function VisualizationPanel({
  dashboardItem,
  isMultiPanelView,
  handleClosePanel,
  dragHandleProps,
  getDashboardItemByType,
  actionButtons,
}: Props): JSX.Element {
  const {flamegraphHint} = useParcaContext();

  return (
    <>
      <div className="flex w-full items-center justify-end gap-2 pb-2">
        <div
          className={cx(
            'flex w-full justify-between flex-col-reverse md:flex-row',
            isMultiPanelView && dashboardItem === 'icicle' ? 'items-end gap-x-2' : 'items-end'
          )}
        >
          <div className="flex items-center gap-2">
            <div
              className={cx(isMultiPanelView ? '' : 'hidden', 'flex items-center')}
              {...dragHandleProps}
            >
              <Icon className="text-xl" icon="material-symbols:drag-indicator" />
            </div>
            {isMultiPanelView ? (
              <div className="flex gap-2">
                {actionButtons[dashboardItem as keyof typeof actionButtons]}
              </div>
            ) : null}
          </div>
          <div
            className={cx(
              'flex flex-row items-center gap-4',
              isMultiPanelView && dashboardItem === 'icicle' && 'pb-[10px]'
            )}
          >
            {dashboardItem === 'icicle' && flamegraphHint != null ? (
              <div className="px-2">{flamegraphHint}</div>
            ) : null}
          </div>
        </div>
        {isMultiPanelView && (
          <IconButton
            className="py-0"
            onClick={() => handleClosePanel(dashboardItem)}
            icon={<CloseIcon />}
          />
        )}
      </div>
      {getDashboardItemByType({
        type: dashboardItem,
        isHalfScreen: isMultiPanelView,
      })}
    </>
  );
});
