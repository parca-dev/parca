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

import {Profiler, ProfilerProps, useEffect, useMemo, useState} from 'react';

import {Icon} from '@iconify/react';
import cx from 'classnames';
import {scaleLinear} from 'd3';
import graphviz from 'graphviz-wasm';
import {
  DragDropContext,
  Draggable,
  Droppable,
  type DraggableLocation,
  type DropResult,
} from 'react-beautiful-dnd';

import {
  Callgraph as CallgraphType,
  Flamegraph,
  FlamegraphArrow,
  QueryServiceClient,
  Source,
  TableArrow,
  Top,
} from '@parca/client';
import {
  Button,
  ConditionalWrapper,
  KeyDownProvider,
  UserPreferences,
  useParcaContext,
  useURLState,
} from '@parca/components';
import {useContainerDimensions} from '@parca/hooks';
import {selectDarkMode, useAppSelector} from '@parca/store';
import {getNewSpanColor} from '@parca/utilities';

import {Callgraph} from '../';
import {jsonToDot} from '../Callgraph/utils';
import ProfileIcicleGraph from '../ProfileIcicleGraph';
import {ProfileSource} from '../ProfileSource';
import {SourceView} from '../SourceView';
import Table from '../Table';
import ProfileShareButton from '../components/ProfileShareButton';
import useDelayedLoader from '../useDelayedLoader';
import FilterByFunctionButton from './FilterByFunctionButton';
import {ProfileViewContextProvider} from './ProfileViewContext';
import ViewSelector from './ViewSelector';
import {VisualizationPanel} from './VisualizationPanel';

type NavigateFunction = (path: string, queryParams: any, options?: {replace?: boolean}) => void;

export interface FlamegraphData {
  loading: boolean;
  data?: Flamegraph;
  arrow?: FlamegraphArrow;
  total?: bigint;
  filtered?: bigint;
  error?: any;
}

export interface TopTableData {
  loading: boolean;
  arrow?: TableArrow;
  data?: Top; // TODO: Remove this once we only have arrow support
  total?: bigint;
  filtered?: bigint;
  error?: any;
}

interface CallgraphData {
  loading: boolean;
  data?: CallgraphType;
  total?: bigint;
  filtered?: bigint;
  error?: any;
}

interface SourceData {
  loading: boolean;
  data?: Source;
  error?: any;
}

export interface ProfileViewProps {
  total: bigint;
  filtered: bigint;
  flamegraphData: FlamegraphData;
  topTableData?: TopTableData;
  callgraphData?: CallgraphData;
  sourceData?: SourceData;
  sampleUnit: string;
  profileSource?: ProfileSource;
  queryClient?: QueryServiceClient;
  navigateTo?: NavigateFunction;
  compare?: boolean;
  onDownloadPProf: () => void;
  pprofDownloading?: boolean;
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
  total,
  filtered,
  flamegraphData,
  topTableData,
  callgraphData,
  sourceData,
  sampleUnit,
  profileSource,
  queryClient,
  navigateTo,
  onDownloadPProf,
  pprofDownloading,
}: ProfileViewProps): JSX.Element => {
  const {ref, dimensions} = useContainerDimensions();
  const [curPath, setCurPath] = useState<string[]>([]);
  const [rawDashboardItems = ['icicle'], setDashboardItems] = useURLState({
    param: 'dashboard_items',
    navigateTo,
  });
  const [graphvizLoaded, setGraphvizLoaded] = useState(false);
  const [callgraphSVG, setCallgraphSVG] = useState<string | undefined>(undefined);
  const [currentSearchString] = useURLState({param: 'search_string'});

  const dashboardItems = useMemo(() => {
    if (rawDashboardItems !== undefined) {
      return rawDashboardItems as string[];
    }
    return ['icicle'];
  }, [rawDashboardItems]);

  const isDarkMode = useAppSelector(selectDarkMode);
  const isMultiPanelView = dashboardItems.length > 1;

  const {loader, perf} = useParcaContext();

  useEffect(() => {
    // Reset the current path when the profile source changes
    setCurPath([]);
  }, [profileSource]);

  useEffect(() => {
    async function loadGraphviz(): Promise<void> {
      await graphviz.loadWASM();
      setGraphvizLoaded(true);
    }
    void loadGraphviz();
  }, []);

  const isLoading = useMemo(() => {
    if (dashboardItems.includes('icicle')) {
      return Boolean(flamegraphData?.loading);
    }
    if (dashboardItems.includes('callgraph')) {
      return Boolean(callgraphData?.loading) || Boolean(callgraphSVG === undefined);
    }
    if (dashboardItems.includes('table')) {
      return Boolean(topTableData?.loading);
    }
    if (dashboardItems.includes('source')) {
      return Boolean(sourceData?.loading);
    }
    return false;
  }, [
    dashboardItems,
    callgraphData?.loading,
    flamegraphData?.loading,
    topTableData?.loading,
    sourceData?.loading,
    callgraphSVG,
  ]);

  const isLoaderVisible = useDelayedLoader(isLoading);

  const maxColor: string = getNewSpanColor(isDarkMode);
  const minColor: string = scaleLinear([isDarkMode ? 'black' : 'white', maxColor])(0.3);
  const colorRange: [string, string] = [minColor, maxColor];
  // Note: If we want to further optimize the experience, we could try to load the graphviz layout in the ProfileViewWithData layer
  // and pass it down to the ProfileView. This would allow us to load the layout in parallel with the flamegraph data.
  // However, the layout calculation is dependent on the width and color range of the graph container, which is why it is done at this level
  useEffect(() => {
    async function loadCallgraphSVG(
      graph: CallgraphType,
      width: number,
      colorRange: [string, string]
    ): Promise<void> {
      await setCallgraphSVG(undefined);
      // Translate JSON to 'dot' graph string
      const dataAsDot = await jsonToDot({
        graph,
        width,
        colorRange,
      });

      // Use Graphviz-WASM to translate the 'dot' graph to a 'JSON' graph
      const svgGraph = await graphviz.layout(dataAsDot, 'svg', 'dot');
      await setCallgraphSVG(svgGraph);
    }

    if (
      graphvizLoaded &&
      callgraphData?.data !== null &&
      callgraphData?.data !== undefined &&
      dimensions?.width !== undefined
    ) {
      void loadCallgraphSVG(callgraphData?.data, dimensions?.width, colorRange);
    }

    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [graphvizLoaded, callgraphData?.data]);

  const setNewCurPath = (path: string[]): void => {
    if (!arrayEquals(curPath, path)) {
      setCurPath(path);
    }
  };

  const getDashboardItemByType = ({
    type,
    isHalfScreen,
    setActionButtons,
  }: {
    type: string;
    isHalfScreen: boolean;
    setActionButtons: (actionButtons: JSX.Element) => void;
  }): JSX.Element => {
    switch (type) {
      case 'icicle': {
        return (
          <ConditionalWrapper<ProfilerProps>
            condition={perf?.onRender != null}
            WrapperComponent={Profiler}
            wrapperProps={{
              id: 'icicleGraph',
              onRender: perf?.onRender as React.ProfilerOnRenderCallback,
            }}
          >
            <ProfileIcicleGraph
              curPath={curPath}
              setNewCurPath={setNewCurPath}
              arrow={flamegraphData?.arrow}
              graph={flamegraphData?.data}
              total={total}
              filtered={filtered}
              sampleUnit={sampleUnit}
              navigateTo={navigateTo}
              loading={flamegraphData.loading}
              setActionButtons={setActionButtons}
              error={flamegraphData.error}
              width={
                dimensions?.width !== undefined
                  ? isHalfScreen
                    ? (dimensions.width - 40) / 2
                    : dimensions.width - 16
                  : 0
              }
            />
          </ConditionalWrapper>
        );
      }
      case 'callgraph': {
        return callgraphData?.data !== undefined &&
          callgraphSVG !== undefined &&
          dimensions?.width !== undefined ? (
          <Callgraph
            data={callgraphData.data}
            svgString={callgraphSVG}
            sampleUnit={sampleUnit}
            width={isHalfScreen ? dimensions?.width / 2 : dimensions?.width}
          />
        ) : (
          <></>
        );
      }
      case 'table': {
        return topTableData != null ? (
          <Table
            total={total}
            filtered={filtered}
            loading={topTableData.loading}
            data={topTableData.arrow?.record}
            sampleUnit={sampleUnit}
            navigateTo={navigateTo}
            setActionButtons={setActionButtons}
            currentSearchString={currentSearchString as string}
          />
        ) : (
          <></>
        );
      }
      case 'source': {
        return sourceData != null ? (
          <SourceView
            loading={sourceData.loading}
            data={sourceData.data}
            total={total}
            filtered={filtered}
            setActionButtons={setActionButtons}
          />
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

  // TODO: this is just a placeholder, we need to replace with an actually informative and accurate title (cc @metalmatze)
  const profileSourceString = profileSource?.toString();
  const hasProfileSource = profileSource !== undefined && profileSourceString !== '';
  const headerParts = profileSourceString?.split('"') ?? [];

  return (
    <KeyDownProvider>
      <ProfileViewContextProvider value={{profileSource, sampleUnit}}>
        <div
          className={cx(
            'mb-4 flex w-full items-center',
            hasProfileSource ? 'justify-between' : 'justify-end'
          )}
        >
          {hasProfileSource && (
            <div className="max-w-[300px]">
              <div className="text-sm font-medium capitalize">
                {headerParts.length > 0 ? headerParts[0].replace(/"/g, '') : ''}
              </div>
              <div className="text-xs">
                {headerParts.length > 1
                  ? headerParts[headerParts.length - 1].replace(/"/g, '')
                  : ''}
              </div>
            </div>
          )}

          <div className="flex items-center md:justify-end gap-2 flex-wrap">
            <FilterByFunctionButton navigateTo={navigateTo} />
            <UserPreferences
              customButton={
                <Button className="gap-2" variant="neutral">
                  Preferences
                  <Icon icon="pajamas:preferences" width={20} />
                </Button>
              }
            />
            {profileSource !== undefined && queryClient !== undefined ? (
              <ProfileShareButton
                queryRequest={profileSource.QueryRequest()}
                queryClient={queryClient}
              />
            ) : null}
            <Button
              className="gap-2"
              variant="neutral"
              onClick={e => {
                e.preventDefault();
                onDownloadPProf();
              }}
              disabled={pprofDownloading}
            >
              {pprofDownloading != null && pprofDownloading ? 'Downloading...' : 'Download pprof'}
              <Icon icon="material-symbols:download" width={20} />
            </Button>
            <ViewSelector
              defaultValue=""
              navigateTo={navigateTo}
              position={-1}
              placeholderText="Add panel"
              icon={<Icon icon="material-symbols:add" width={20} />}
              addView={true}
              disabled={isMultiPanelView || dashboardItems.length < 1}
            />
          </div>
        </div>

        <div className="w-full" ref={ref}>
          {isLoaderVisible ? (
            <>{loader}</>
          ) : (
            <DragDropContext onDragEnd={onDragEnd}>
              <Droppable droppableId="droppable" direction="horizontal">
                {provided => (
                  <div
                    ref={provided.innerRef}
                    className={cx(
                      'grid w-full gap-2',
                      isMultiPanelView ? 'grid-cols-2' : 'grid-cols-1'
                    )}
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
                                'min-h-[200px] w-full rounded p-2 shadow dark:border dark:border-gray-700 dark:bg-gray-700',
                                snapshot.isDragging
                                  ? 'bg-gray-200 dark:bg-gray-500'
                                  : 'bg-white dark:bg-gray-700'
                              )}
                            >
                              <VisualizationPanel
                                handleClosePanel={handleClosePanel}
                                isMultiPanelView={isMultiPanelView}
                                dashboardItem={dashboardItem}
                                getDashboardItemByType={getDashboardItemByType}
                                dragHandleProps={provided.dragHandleProps}
                                navigateTo={navigateTo}
                                index={index}
                              />
                            </div>
                          )}
                        </Draggable>
                      );
                    })}
                  </div>
                )}
              </Droppable>
            </DragDropContext>
          )}
        </div>
      </ProfileViewContextProvider>
    </KeyDownProvider>
  );
};
