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

import {Profiler, ProfilerProps, useCallback, useEffect, useState} from 'react';

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
import {ConditionalWrapper, KeyDownProvider, useParcaContext, useURLState} from '@parca/components';
import {useContainerDimensions} from '@parca/hooks';
import {selectDarkMode, useAppSelector} from '@parca/store';
import {getNewSpanColor, selectQueryParam} from '@parca/utilities';

import {Callgraph} from '../';
import {jsonToDot} from '../Callgraph/utils';
import ProfileIcicleGraph from '../ProfileIcicleGraph';
import {FIELD_FUNCTION_NAME} from '../ProfileIcicleGraph/IcicleGraphArrow';
import {ProfileSource} from '../ProfileSource';
import {SourceView} from '../SourceView';
import {Table} from '../Table';
import VisualisationToolbar from '../components/VisualisationToolbar';
import {ProfileViewContextProvider} from './ProfileViewContext';
import {VisualizationPanel} from './VisualizationPanel';

export interface FlamegraphData {
  loading: boolean;
  data?: Flamegraph;
  arrow?: FlamegraphArrow;
  total?: bigint;
  filtered?: bigint;
  error?: any;
  mappings?: string[];
  mappingsLoading: boolean;
}

export interface TopTableData {
  loading: boolean;
  arrow?: TableArrow;
  data?: Top; // TODO: Remove this once we only have arrow support
  total?: bigint;
  filtered?: bigint;
  error?: any;
  unit?: string;
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
  profileSource?: ProfileSource;
  queryClient?: QueryServiceClient;
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
  profileSource,
  queryClient,
  onDownloadPProf,
  pprofDownloading,
  compare,
}: ProfileViewProps): JSX.Element => {
  const {timezone} = useParcaContext();
  const {ref, dimensions} = useContainerDimensions();
  const [curPath, setCurPath] = useState<string[]>([]);
  const [dashboardItems, setDashboardItems] = useURLState<string[]>('dashboard_items', {
    alwaysReturnArray: true,
  });
  const [graphvizLoaded, setGraphvizLoaded] = useState(false);
  const [callgraphSVG, setCallgraphSVG] = useState<string | undefined>(undefined);
  const [currentSearchString, setSearchString] = useURLState<string | undefined>('search_string');

  const isDarkMode = useAppSelector(selectDarkMode);
  const isMultiPanelView = dashboardItems.length > 1;

  const {perf, profileViewExternalMainActions} = useParcaContext();

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
              profileType={profileSource?.ProfileType()}
              loading={flamegraphData.loading}
              setActionButtons={setActionButtons}
              error={flamegraphData.error}
              isHalfScreen={isHalfScreen}
              width={
                dimensions?.width !== undefined
                  ? isHalfScreen
                    ? (dimensions.width - 40) / 2
                    : dimensions.width - 16
                  : 0
              }
              mappings={flamegraphData.mappings}
              mappingsLoading={flamegraphData.mappingsLoading}
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
            profileType={profileSource?.ProfileType()}
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
            unit={topTableData.unit}
            profileType={profileSource?.ProfileType()}
            setActionButtons={setActionButtons}
            currentSearchString={currentSearchString}
            setSearchString={setSearchString}
            isHalfScreen={isHalfScreen}
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
  const profileSourceString = profileSource?.toString(timezone);
  const hasProfileSource = profileSource !== undefined && profileSourceString !== '';
  const headerParts = profileSourceString?.split('"') ?? [];

  const compareMode =
    compare === true ||
    (selectQueryParam('compare_a') === 'true' && selectQueryParam('compare_b') === 'true');

  const [groupBy, setStoreGroupBy] = useURLState<string[]>('group_by', {
    defaultValue: [FIELD_FUNCTION_NAME],
    alwaysReturnArray: true,
  });

  const setGroupBy = useCallback(
    (keys: string[]): void => {
      setStoreGroupBy(keys);
    },
    [setStoreGroupBy]
  );

  const toggleGroupBy = useCallback(
    (key: string): void => {
      groupBy.includes(key)
        ? setGroupBy(groupBy.filter(v => v !== key)) // remove
        : setGroupBy([...groupBy, key]); // add
    },
    [groupBy, setGroupBy]
  );

  return (
    <KeyDownProvider>
      <ProfileViewContextProvider value={{profileSource, compareMode}}>
        <div className="border-t border-gray-200 dark:border-gray-700 h-[1px] w-full pb-4"></div>
        <div
          className={cx(
            'flex w-full',
            hasProfileSource || profileViewExternalMainActions != null
              ? 'justify-center'
              : 'justify-end',
            {
              'items-end mb-4': !hasProfileSource && profileViewExternalMainActions != null,
              'items-center mb-2': hasProfileSource,
            }
          )}
        >
          <div>
            {hasProfileSource && (
              <div className="flex items-center gap-1">
                <div className="text-xs font-medium">
                  {headerParts.length > 0 ? headerParts[0].replace(/"/g, '') : ''}
                </div>
                <div className="text-xs font-medium">
                  {headerParts.length > 1
                    ? headerParts[headerParts.length - 1].replace(/"/g, '')
                    : ''}
                </div>
              </div>
            )}

            {profileViewExternalMainActions != null ? profileViewExternalMainActions : null}
          </div>
        </div>

        <VisualisationToolbar
          groupBy={groupBy}
          toggleGroupBy={toggleGroupBy}
          hasProfileSource={hasProfileSource}
          pprofdownloading={pprofDownloading}
          profileSource={profileSource}
          queryClient={queryClient}
          onDownloadPProf={onDownloadPProf}
          isMultiPanelView={isMultiPanelView}
          dashboardItems={dashboardItems}
          curPath={curPath}
          setNewCurPath={setNewCurPath}
          profileType={profileSource?.ProfileType()}
          total={total}
          filtered={filtered}
          currentSearchString={currentSearchString}
          setSearchString={setSearchString}
        />

        <div className="w-full" ref={ref}>
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
                              'w-full min-h-96',
                              snapshot.isDragging
                                ? 'bg-gray-200 dark:bg-gray-500'
                                : 'bg-white dark:bg-gray-900'
                            )}
                          >
                            <VisualizationPanel
                              handleClosePanel={handleClosePanel}
                              isMultiPanelView={isMultiPanelView}
                              dashboardItem={dashboardItem}
                              getDashboardItemByType={getDashboardItemByType}
                              dragHandleProps={provided.dragHandleProps}
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
        </div>
      </ProfileViewContextProvider>
    </KeyDownProvider>
  );
};
