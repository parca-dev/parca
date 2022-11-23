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
import {NavigateOptions} from 'react-router-dom';
import useUIFeatureFlag from '@parca/functions/useUIFeatureFlag';
import {QueryServiceClient, Flamegraph, Top, Callgraph as CallgraphType} from '@parca/client';
import {Button, Card, SearchNodes, useParcaContext} from '@parca/components';
import {useContainerDimensions} from '@parca/dynamicsize';
import {
  useAppSelector,
  selectDarkMode,
  selectDashboardItems,
  setDashboardItems,
  DashboardItem,
  useAppDispatch,
} from '@parca/store';

import {Callgraph} from '../';
import ProfileShareButton from '../components/ProfileShareButton';
import FilterByFunctionButton from './FilterByFunctionButton';
import ProfileIcicleGraph from '../ProfileIcicleGraph';
import {ProfileSource} from '../ProfileSource';
import TopTable from '../TopTable';
import useDelayedLoader from '../useDelayedLoader';

import '../ProfileView.styles.css';

type NavigateFunction = (path: string, queryParams: any, options?: NavigateOptions) => void;

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
}: ProfileViewProps): JSX.Element => {
  const {ref, dimensions} = useContainerDimensions();
  const [curPath, setCurPath] = useState<string[]>([]);

  const dispatch = useAppDispatch();

  const isDarkMode = useAppSelector(selectDarkMode);

  const dashboardItems = useAppSelector(selectDashboardItems);
  const [callgraphEnabled] = useUIFeatureFlag('callgraph');
  const [filterByFunctionEnabled] = useUIFeatureFlag('filterByFunction');

  const {loader, perf} = useParcaContext();

  useEffect(() => {
    // Reset the current path when the profile source changes
    setCurPath([]);
  }, [profileSource]);

  // every time the dashboardItems changes in store, we need to navigate to new URL
  useEffect(() => {
    const router = parseParams(window.location.search);
    if (navigateTo != null) {
      navigateTo('/', {
        ...router,
        ...{dashboard_items: encodeURIComponent(dashboardItems.join(','))},
      });
    }
  }, [dashboardItems]);

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

  if (isLoaderVisible) {
    return <>{loader}</>;
  }

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

  const switchDashboardItems = (dashboardItems: DashboardItem[]): void => {
    dispatch(setDashboardItems(dashboardItems));
  };

  const maxColor: string = getNewSpanColor(isDarkMode);
  const minColor: string = scaleLinear([isDarkMode ? 'black' : 'white', maxColor])(0.3);
  const colorRange: [string, string] = [minColor, maxColor];

  const isSinglePanelView = dashboardItems.length === 1;

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
                    />
                  ) : null}

                  <Button
                    color="neutral"
                    onClick={e => {
                      e.preventDefault();
                      onDownloadPProf();
                    }}
                  >
                    Download pprof
                  </Button>
                </div>
                {filterByFunctionEnabled ? (
                  <FilterByFunctionButton navigateTo={navigateTo} />
                ) : (
                  <SearchNodes navigateTo={navigateTo} />
                )}
              </div>

              <div className="flex ml-auto gap-2">
                {filterByFunctionEnabled ? <SearchNodes navigateTo={navigateTo} /> : null}
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
                    variant={`${dashboardItems.includes('callgraph') ? 'primary' : 'neutral'}`}
                    onClick={() => switchDashboardItems(['callgraph'])}
                    className="whitespace-nowrap text-ellipsis"
                  >
                    Callgraph
                  </Button>
                ) : null}

                <div className="flex">
                  <Button
                    variant={`${dashboardItems.includes('table') ? 'primary' : 'neutral'}`}
                    className="items-center rounded-tr-none rounded-br-none w-auto px-8 whitespace-nowrap text-ellipsis no-outline-on-buttons"
                    onClick={() => switchDashboardItems(['table'])}
                  >
                    Table
                  </Button>

                  <Button
                    variant={`${
                      dashboardItems.includes('table') && dashboardItems.includes('icicle')
                        ? 'primary'
                        : 'neutral'
                    }`}
                    className="items-center rounded-tl-none rounded-tr-none rounded-bl-none rounded-br-none border-l-0 border-r-0 w-auto px-8 whitespace-nowrap no-outline-on-buttons text-ellipsis"
                    onClick={() => switchDashboardItems(['table', 'icicle'])}
                  >
                    Both
                  </Button>

                  <Button
                    variant={`${dashboardItems.includes('icicle') ? 'primary' : 'neutral'}`}
                    className="items-center rounded-tl-none rounded-bl-none w-auto px-8 whitespace-nowrap text-ellipsis no-outline-on-buttons"
                    onClick={() => switchDashboardItems(['icicle'])}
                  >
                    Icicle Graph
                  </Button>
                </div>
              </div>
            </div>

            {isSinglePanelView && (
              <div ref={ref} className="flex space-x-4 justify-between w-full">
                {dashboardItems.includes('icicle') && flamegraphData?.data != null && (
                  <div className="w-full">
                    <Profiler id="icicleGraph" onRender={perf.onRender}>
                      <ProfileIcicleGraph
                        curPath={curPath}
                        setNewCurPath={setNewCurPath}
                        graph={flamegraphData.data}
                        sampleUnit={sampleUnit}
                      />
                    </Profiler>
                  </div>
                )}
                {dashboardItems.includes('callgraph') && callgraphData?.data != null && (
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
                {dashboardItems.includes('table') && topTableData != null && (
                  <div className="w-full">
                    <TopTable
                      data={topTableData.data}
                      sampleUnit={sampleUnit}
                      navigateTo={navigateTo}
                    />
                  </div>
                )}
              </div>
            )}
            {!isSinglePanelView && (
              <div ref={ref} className="flex space-x-4 justify-between w-full">
                {dashboardItems.includes('icicle') && dashboardItems.includes('table') && (
                  <>
                    <div className="w-1/2">
                      <TopTable
                        data={topTableData?.data}
                        sampleUnit={sampleUnit}
                        navigateTo={navigateTo}
                      />
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
