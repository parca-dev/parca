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

import {Profiler, useEffect, useMemo, useState} from 'react';
import {scaleLinear} from 'd3';

import cx from 'classnames';
import {getNewSpanColor, useURLState} from '@parca/functions';
import {CloseIcon} from '@parca/icons';
import {Icon} from '@iconify/react';
import {QueryServiceClient, Flamegraph, Top, Callgraph as CallgraphType} from '@parca/client';
import {Button, Card, useParcaContext} from '@parca/components';
import {useContainerDimensions} from '@parca/dynamicsize';
import {useAppSelector, selectDarkMode} from '@parca/store';
import {
  DragDropContext,
  Droppable,
  Draggable,
  DropResult,
  DraggableLocation,
} from 'react-beautiful-dnd';

import {Callgraph} from '../';
import ProfileShareButton from '../components/ProfileShareButton';
import FilterByFunctionButton from './FilterByFunctionButton';
import ViewSelector from './ViewSelector';
import ProfileIcicleGraph, {ResizeHandler} from '../ProfileIcicleGraph';
import {ProfileSource} from '../ProfileSource';
import TopTable from '../TopTable';
import useDelayedLoader from '../useDelayedLoader';

import '../ProfileView.styles.css';

type NavigateFunction = (path: string, queryParams: any, options?: {replace?: boolean}) => void;

export interface FlamegraphData {
  loading: boolean;
  data?: Flamegraph;
  error?: any;
}

export interface TopTableData {
  loading: boolean;
  data?: Top;
  error?: any;
}

interface CallgraphData {
  loading: boolean;
  data?: CallgraphType;
  error?: any;
}

export interface ProfileViewProps {
  flamegraphData?: FlamegraphData;
  topTableData?: TopTableData;
  callgraphData?: CallgraphData;
  sampleUnit: string;
  profileSource?: ProfileSource;
  queryClient?: QueryServiceClient;
  navigateTo?: NavigateFunction;
  compare?: boolean;
  onDownloadPProf: () => void;
  onFlamegraphContainerResize?: ResizeHandler;
}

function arrayEquals<T>(a: T[], b: T[]): boolean {
  return (
    Array.isArray(a) &&
    Array.isArray(b) &&
    a.length === b.length &&
    a.every((val, index) => val === b[index])
  );
}

export const ProfileView = ({
  flamegraphData,
  topTableData,
  callgraphData,
  sampleUnit,
  profileSource,
  queryClient,
  navigateTo,
  onDownloadPProf,
  onFlamegraphContainerResize,
}: ProfileViewProps): JSX.Element => {
  const {ref, dimensions} = useContainerDimensions();
  const [curPath, setCurPath] = useState<string[]>([]);
  const [rawDashboardItems, setDashboardItems] = useURLState({
    param: 'dashboard_items',
    navigateTo,
  });
  const dashboardItems = rawDashboardItems as string[];
  const isDarkMode = useAppSelector(selectDarkMode);
  const isMultiPanelView = dashboardItems.length > 1;

  const {loader, perf} = useParcaContext();

  useEffect(() => {
    // Reset the current path when the profile source changes
    setCurPath([]);
  }, [profileSource]);

  const isLoading = useMemo(() => {
    if (dashboardItems.includes('icicle')) {
      return Boolean(flamegraphData?.loading);
    }
    if (dashboardItems.includes('callgraph')) {
      return Boolean(callgraphData?.loading);
    }
    if (dashboardItems.includes('table')) {
      return Boolean(topTableData?.loading);
    }
    return false;
  }, [dashboardItems, callgraphData?.loading, flamegraphData?.loading, topTableData?.loading]);

  const isLoaderVisible = useDelayedLoader(isLoading);

  if (flamegraphData?.error != null) {
    console.error('Error: ', flamegraphData?.error);
    return (
      <div className="p-10 flex justify-center">
        An error occurred: {flamegraphData?.error.message}
      </div>
    );
  }

  const setNewCurPath = (path: string[]): void => {
    if (!arrayEquals(curPath, path)) {
      setCurPath(path);
    }
  };

  const maxColor: string = getNewSpanColor(isDarkMode);
  const minColor: string = scaleLinear([isDarkMode ? 'black' : 'white', maxColor])(0.3);
  const colorRange: [string, string] = [minColor, maxColor];

  const getDashboardItemByType = ({
    type,
    isHalfScreen,
  }: {
    type: string;
    isHalfScreen: boolean;
  }): JSX.Element => {
    switch (type) {
      case 'icicle': {
        return flamegraphData?.data != null ? (
          <Profiler id="icicleGraph" onRender={perf?.onRender as React.ProfilerOnRenderCallback}>
            <ProfileIcicleGraph
              curPath={curPath}
              setNewCurPath={setNewCurPath}
              graph={flamegraphData.data}
              sampleUnit={sampleUnit}
              onContainerResize={onFlamegraphContainerResize}
            />
          </Profiler>
        ) : (
          <></>
        );
      }
      case 'callgraph': {
        return callgraphData?.data != null && dimensions?.width !== undefined ? (
          <Callgraph
            graph={callgraphData.data}
            sampleUnit={sampleUnit}
            width={isHalfScreen ? dimensions?.width / 2 : dimensions?.width}
            colorRange={colorRange}
          />
        ) : (
          <></>
        );
      }
      case 'table': {
        return topTableData != null ? (
          <TopTable data={topTableData.data} sampleUnit={sampleUnit} navigateTo={navigateTo} />
        ) : (
          <></>
        );
      }
      default: {
        return <></>;
      }
    }
  };

  const handleClosePanel = (visualizationType: string): void => {
    const newDashboardItems = dashboardItems.filter(item => item !== visualizationType);
    setDashboardItems(newDashboardItems);
  };

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
    <>
      <div className="py-3">
        <Card>
          <Card.Body>
            <div className="flex py-3 w-full">
              <div className="w-2/5 flex space-x-4">
                <div className="flex space-x-1">
                  {profileSource != null && queryClient != null ? (
                    <ProfileShareButton
                      queryRequest={profileSource.QueryRequest()}
                      queryClient={queryClient}
                      disabled={isLoading}
                    />
                  ) : null}

                  <Button
                    color="neutral"
                    onClick={e => {
                      e.preventDefault();
                      onDownloadPProf();
                    }}
                    disabled={isLoading}
                  >
                    Download pprof
                  </Button>
                </div>
                <FilterByFunctionButton navigateTo={navigateTo} />
              </div>

              <div className="flex ml-auto gap-2">
                <ViewSelector
                  defaultValue=""
                  navigateTo={navigateTo}
                  position={-1}
                  placeholderText="Add panel..."
                  primary
                  addView={true}
                  disabled={isMultiPanelView || dashboardItems.length < 1}
                />
              </div>
            </div>

            {isLoaderVisible ? (
              <>{loader}</>
            ) : (
              <DragDropContext onDragEnd={onDragEnd}>
                <div className="w-full" ref={ref}>
                  <Droppable droppableId="droppable" direction="horizontal">
                    {provided => (
                      <div
                        ref={provided.innerRef}
                        className="flex space-x-4 justify-between w-full"
                        {...provided.droppableProps}
                      >
                        {dashboardItems.map((dashboardItem, index) => {
                          return (
                            <Draggable
                              key={dashboardItem}
                              draggableId={dashboardItem}
                              index={index}
                              isDragDisabled={!isMultiPanelView}
                            >
                              {(provided, snapshot: {isDragging: boolean}) => (
                                <div
                                  ref={provided.innerRef}
                                  {...provided.draggableProps}
                                  key={dashboardItem}
                                  className={cx(
                                    'border dark:bg-gray-700 rounded border-gray-300 dark:border-gray-500 p-3',
                                    isMultiPanelView ? 'w-1/2' : 'w-full',
                                    snapshot.isDragging ? 'bg-gray-200' : 'bg-white'
                                  )}
                                >
                                  <div className="w-full flex justify-end pb-2">
                                    <div className="w-full flex justify-between">
                                      <div
                                        className={cx(isMultiPanelView ? 'visible' : 'invisible')}
                                        {...provided.dragHandleProps}
                                      >
                                        <Icon
                                          className="text-xl"
                                          icon="material-symbols:drag-indicator"
                                        />
                                      </div>
                                      <ViewSelector
                                        defaultValue={dashboardItem}
                                        navigateTo={navigateTo}
                                        position={index}
                                      />
                                    </div>

                                    {isMultiPanelView && (
                                      <button
                                        type="button"
                                        onClick={() => handleClosePanel(dashboardItem)}
                                        className="pl-2"
                                      >
                                        <CloseIcon />
                                      </button>
                                    )}
                                  </div>
                                  {getDashboardItemByType({
                                    type: dashboardItem,
                                    isHalfScreen: isMultiPanelView,
                                  })}
                                </div>
                              )}
                            </Draggable>
                          );
                        })}
                      </div>
                    )}
                  </Droppable>
                </div>
              </DragDropContext>
            )}
          </Card.Body>
        </Card>
      </div>
    </>
  );
};
