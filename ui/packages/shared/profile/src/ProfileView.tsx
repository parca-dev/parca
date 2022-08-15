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

import React, {useEffect, useMemo, useState} from 'react';

import {parseParams} from '@parca/functions';
import {QueryServiceClient, Flamegraph, Top} from '@parca/client';
import {Button, Card, SearchNodes, useGrpcMetadata, useParcaTheme} from '@parca/components';

import ProfileShareButton from './components/ProfileShareButton';
import ProfileIcicleGraph from './ProfileIcicleGraph';
import {ProfileSource} from './ProfileSource';
import TopTable from './TopTable';
import useDelayedLoader from './useDelayedLoader';
import {downloadPprof} from './utils';

import './ProfileView.styles.css';

type NavigateFunction = (path: string, queryParams: any) => void;

interface FlamegraphData {
  loading: boolean;
  data?: Flamegraph;
  error?: any;
}

interface TopTableData {
  loading: boolean;
  data?: Top;
  error?: any;
}

type VisualizationType = 'icicle' | 'table' | 'both';

interface ProfileVisState {
  currentView: VisualizationType;
  setCurrentView: (view: VisualizationType) => void;
}

interface ProfileViewProps {
  flamegraphData?: FlamegraphData;
  topTableData?: TopTableData;
  sampleUnit: string;
  profileVisState: ProfileVisState;
  profileSource?: ProfileSource;
  queryClient?: QueryServiceClient;
  navigateTo?: NavigateFunction;
  compare?: boolean;
}

function arrayEquals(a, b): boolean {
  return (
    Array.isArray(a) &&
    Array.isArray(b) &&
    a.length === b.length &&
    a.every((val, index) => val === b[index])
  );
}
export const useProfileVisState = (): ProfileVisState => {
  const router = parseParams(window.location.search);
  const currentViewFromURL = router.currentProfileView as string;
  const [currentView, setCurrentView] = useState<VisualizationType>(
    (currentViewFromURL as VisualizationType) || 'icicle'
  );

  return {currentView, setCurrentView};
};

export const ProfileView = ({
  flamegraphData,
  topTableData,
  sampleUnit,
  profileSource,
  queryClient,
  navigateTo,
  profileVisState,
}: ProfileViewProps): JSX.Element => {
  const [curPath, setCurPath] = useState<string[]>([]);
  const {currentView, setCurrentView} = profileVisState;

  const metadata = useGrpcMetadata();
  const {loader} = useParcaTheme();

  useEffect(() => {
    // Reset the current path when the profile source changes
    setCurPath([]);
  }, [profileSource]);

  const isLoading = useMemo(() => {
    if (currentView === 'icicle') {
      return !!flamegraphData?.loading;
    }
    if (currentView === 'table') {
      return !!topTableData?.loading;
    }
    if (currentView === 'both') {
      return !!flamegraphData?.loading || !!topTableData?.loading;
    }
    return false;
  }, [currentView, flamegraphData?.loading, topTableData?.loading]);

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

  const downloadPProf = async (e: React.MouseEvent<HTMLElement>) => {
    e.preventDefault();
    if (profileSource == null || queryClient == null) {
      return;
    }

    try {
      const blob = await downloadPprof(profileSource.QueryRequest(), queryClient, metadata);
      const link = document.createElement('a');
      link.href = window.URL.createObjectURL(blob);
      link.download = 'profile.pb.gz';
      link.click();
    } catch (error) {
      console.error('Error while querying', error);
    }
  };

  const resetIcicleGraph = () => setCurPath([]);

  const setNewCurPath = (path: string[]) => {
    if (!arrayEquals(curPath, path)) {
      setCurPath(path);
    }
  };

  const switchProfileView = (view: VisualizationType) => {
    if (view == null) {
      return;
    }
    if (navigateTo === undefined) return;

    setCurrentView(view);
    const router = parseParams(window.location.search);

    navigateTo('/', {...router, ...{currentProfileView: view}});
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
                    />
                  ) : null}

                  <Button color="neutral" onClick={downloadPProf}>
                    Download pprof
                  </Button>
                </div>

                <SearchNodes />
              </div>

              <div className="flex ml-auto">
                <div className="mr-3">
                  <Button
                    color="neutral"
                    onClick={resetIcicleGraph}
                    disabled={curPath.length === 0}
                    className="whitespace-nowrap text-ellipsis"
                  >
                    Reset View
                  </Button>
                </div>

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

            <div className="flex space-x-4 justify-between">
              {currentView === 'icicle' && flamegraphData?.data != null && (
                <div className="w-full">
                  <ProfileIcicleGraph
                    curPath={curPath}
                    setNewCurPath={setNewCurPath}
                    graph={flamegraphData.data}
                    sampleUnit={sampleUnit}
                  />
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
          </Card.Body>
        </Card>
      </div>
    </>
  );
};
