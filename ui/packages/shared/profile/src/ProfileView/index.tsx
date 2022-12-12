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

import {getNewSpanColor, parseParams} from '@parca/functions';
import useUIFeatureFlag from '@parca/functions/useUIFeatureFlag';
import {QueryServiceClient, Flamegraph, Top, Callgraph as CallgraphType} from '@parca/client';
import {Button, Card, useParcaContext} from '@parca/components';
import {useContainerDimensions} from '@parca/dynamicsize';
import {
  useAppSelector,
  selectDarkMode,
  selectSearchNodeString,
  selectFilterByFunction,
  useAppDispatch,
  setSearchNodeString,
} from '@parca/store';

import {Callgraph} from '../';
import ProfileShareButton from '../components/ProfileShareButton';
import FilterByFunctionButton from './FilterByFunctionButton';
import ProfileIcicleGraph from '../ProfileIcicleGraph';
import {ProfileSource} from '../ProfileSource';
import TopTable from '../TopTable';
import useDelayedLoader from '../useDelayedLoader';

import '../ProfileView.styles.css';

type NavigateFunction = (path: string, queryParams: any) => void;

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

export type VisualizationType = 'icicle' | 'table' | 'callgraph' | 'both';

export interface ProfileVisState {
  currentView: VisualizationType;
  setCurrentView: (view: VisualizationType) => void;
}

export interface ProfileViewProps {
  flamegraphData?: FlamegraphData;
  topTableData?: TopTableData;
  callgraphData?: CallgraphData;
  sampleUnit: string;
  profileVisState: ProfileVisState;
  profileSource?: ProfileSource;
  queryClient?: QueryServiceClient;
  navigateTo?: NavigateFunction;
  compare?: boolean;
  onDownloadPProf: () => void;
}

function arrayEquals<T>(a: T[], b: T[]): boolean {
  return (
    Array.isArray(a) &&
    Array.isArray(b) &&
    a.length === b.length &&
    a.every((val, index) => val === b[index])
  );
}
export const useProfileVisState = (): ProfileVisState => {
  const [currentView, setCurrentView] = useState<VisualizationType>(() => {
    if (typeof window === 'undefined') {
      return 'icicle';
    }
    const router = parseParams(window.location.search);
    const currentViewFromURL = router.currentProfileView as string;

    if (currentViewFromURL != null) {
      return currentViewFromURL as VisualizationType;
    }
    return 'icicle';
  });

  return {currentView, setCurrentView};
};

export const ProfileView = ({
  flamegraphData,
  topTableData,
  callgraphData,
  sampleUnit,
  profileSource,
  queryClient,
  navigateTo,
  profileVisState,
  onDownloadPProf,
}: ProfileViewProps): JSX.Element => {
  const dispatch = useAppDispatch();
  const {ref, dimensions} = useContainerDimensions();
  const [curPath, setCurPath] = useState<string[]>([]);
  const {currentView, setCurrentView} = profileVisState;
  const isDarkMode = useAppSelector(selectDarkMode);
  const currentSearchString = useAppSelector(selectSearchNodeString);
  const filterByFunctionString = useAppSelector(selectFilterByFunction);

  const [callgraphEnabled] = useUIFeatureFlag('callgraph');
  const [highlightAfterFilteringEnabled] = useUIFeatureFlag('highlightAfterFiltering');

  const {loader, perf} = useParcaContext();

  useEffect(() => {
    // Reset the current path when the profile source changes
    setCurPath([]);
  }, [profileSource]);

  const isLoading = useMemo(() => {
    if (currentView === 'icicle') {
      return Boolean(flamegraphData?.loading);
    }
    if (currentView === 'callgraph') {
      return Boolean(callgraphData?.loading);
    }
    if (currentView === 'table') {
      return Boolean(topTableData?.loading);
    }
    if (currentView === 'both') {
      return Boolean(flamegraphData?.loading) || Boolean(topTableData?.loading);
    }
    return false;
  }, [currentView, callgraphData?.loading, flamegraphData?.loading, topTableData?.loading]);

  useEffect(() => {
    if (!highlightAfterFilteringEnabled) {
      if (currentSearchString !== undefined && currentSearchString !== '') {
        dispatch(setSearchNodeString(''));
      }
      return;
    }
    if (isLoading) {
      return;
    }
    if (filterByFunctionString === currentSearchString) {
      return;
    }
    dispatch(setSearchNodeString(filterByFunctionString));
  }, [
    isLoading,
    filterByFunctionString,
    dispatch,
    highlightAfterFilteringEnabled,
    currentSearchString,
  ]);

  const isLoaderVisible = useDelayedLoader(isLoading);

  if (flamegraphData?.error != null) {
    console.error('Error: ', flamegraphData?.error);
    return (
      <div className="p-10 flex justify-center">
        An error occurred: {flamegraphData?.error.message}
      </div>
    );
  }

  const resetIcicleGraph = (): void => setCurPath([]);

  const setNewCurPath = (path: string[]): void => {
    if (!arrayEquals(curPath, path)) {
      setCurPath(path);
    }
  };

  const switchProfileView = (view: VisualizationType): void => {
    if (view == null) {
      return;
    }
    setCurrentView(view);

    if (navigateTo === undefined) {
      return;
    }
    const router = parseParams(window.location.search);
    navigateTo('/', {
      ...router,
      ...{currentProfileView: view},
      ...{searchString: currentSearchString},
    });
  };

  const maxColor: string = getNewSpanColor(isDarkMode);
  // TODO: fix colors for dark mode
  const minColor: string = scaleLinear([isDarkMode ? 'black' : 'white', maxColor])(0.3);

  const colorRange: [string, string] = [minColor, maxColor];

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
                <FilterByFunctionButton />
              </div>

              <div className="flex ml-auto gap-2">
                <Button
                  color="neutral"
                  onClick={resetIcicleGraph}
                  disabled={curPath.length === 0}
                  className="whitespace-nowrap text-ellipsis"
                >
                  Reset View
                </Button>

                {callgraphEnabled ? (
                  <Button
                    variant={`${currentView === 'callgraph' ? 'primary' : 'neutral'}`}
                    onClick={() => switchProfileView('callgraph')}
                    className="whitespace-nowrap text-ellipsis"
                  >
                    Callgraph
                  </Button>
                ) : null}

                <div className="flex">
                  <Button
                    variant={`${currentView === 'table' ? 'primary' : 'neutral'}`}
                    className="items-center rounded-tr-none rounded-br-none w-auto px-8 whitespace-nowrap text-ellipsis no-outline-on-buttons"
                    onClick={() => switchProfileView('table')}
                  >
                    Table
                  </Button>

                  <Button
                    variant={`${currentView === 'both' ? 'primary' : 'neutral'}`}
                    className="items-center rounded-tl-none rounded-tr-none rounded-bl-none rounded-br-none border-l-0 border-r-0 w-auto px-8 whitespace-nowrap no-outline-on-buttons text-ellipsis"
                    onClick={() => switchProfileView('both')}
                  >
                    Both
                  </Button>

                  <Button
                    variant={`${currentView === 'icicle' ? 'primary' : 'neutral'}`}
                    className="items-center rounded-tl-none rounded-bl-none w-auto px-8 whitespace-nowrap text-ellipsis no-outline-on-buttons"
                    onClick={() => switchProfileView('icicle')}
                  >
                    Icicle Graph
                  </Button>
                </div>
              </div>
            </div>

            {isLoaderVisible ? (
              <>{loader}</>
            ) : (
              <div ref={ref} className="flex space-x-4 justify-between w-full">
                {currentView === 'icicle' && flamegraphData?.data != null && (
                  <div className="w-full">
                    <Profiler
                      id="icicleGraph"
                      onRender={perf?.onRender as React.ProfilerOnRenderCallback}
                    >
                      <ProfileIcicleGraph
                        curPath={curPath}
                        setNewCurPath={setNewCurPath}
                        graph={flamegraphData.data}
                        sampleUnit={sampleUnit}
                      />
                    </Profiler>
                  </div>
                )}
                {currentView === 'callgraph' && callgraphData?.data != null && (
                  <div className="w-full">
                    {dimensions?.width !== undefined && (
                      <Callgraph
                        graph={callgraphData.data}
                        sampleUnit={sampleUnit}
                        width={dimensions?.width}
                        colorRange={colorRange}
                      />
                    )}
                  </div>
                )}
                {currentView === 'table' && topTableData != null && (
                  <div className="w-full">
                    <TopTable data={topTableData.data} sampleUnit={sampleUnit} />
                  </div>
                )}
                {currentView === 'both' && (
                  <>
                    <div className="w-1/2">
                      <TopTable data={topTableData?.data} sampleUnit={sampleUnit} />
                    </div>

                    <div className="w-1/2">
                      {flamegraphData != null && (
                        <ProfileIcicleGraph
                          curPath={curPath}
                          setNewCurPath={setNewCurPath}
                          graph={flamegraphData.data}
                          sampleUnit={sampleUnit}
                        />
                      )}
                    </div>
                  </>
                )}
              </div>
            )}
          </Card.Body>
        </Card>
      </div>
    </>
  );
};
