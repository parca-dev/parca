import React, {useEffect, useState} from 'react';
import {CalcWidth} from '@parca/dynamicsize';
import {parseParams} from '@parca/functions';
import {QueryServiceClient, QueryResponse, QueryRequest_ReportType} from '@parca/client';
import {RpcError} from '@protobuf-ts/runtime-rpc';
import {Button, Card, Spinner, useGrpcMetadata} from '@parca/components';
import * as parca_query_v1alpha1_query_pb from '@parca/client/src/parca/query/v1alpha1/query_pb';

import ProfileIcicleGraph from './ProfileIcicleGraph';
import {ProfileSource} from './ProfileSource';
import {useQuery} from './useQuery';
import TopTable from './TopTable';

import './ProfileView.styles.css';

type NavigateFunction = (path: string, queryParams: any) => void;

interface ProfileViewProps {
  queryClient: QueryServiceClient;
  profileSource: ProfileSource;
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

export const ProfileView = ({
  queryClient,
  profileSource,
  navigateTo,
}: ProfileViewProps): JSX.Element => {
  const router = parseParams(window.location.search);
  const currentViewFromURL = router.currentProfileView as string;
  const [curPath, setCurPath] = useState<string[]>([]);
  const [isLoaderVisible, setIsLoaderVisible] = useState<boolean>(false);
  const {isLoading, response, error} = useQuery(
    queryClient,
    profileSource,
    QueryRequest_ReportType.FLAMEGRAPH_UNSPECIFIED
  );
  const [currentView, setCurrentView] = useState<string | undefined>(currentViewFromURL);
  const metadata = useGrpcMetadata();

  useEffect(() => {
    let showLoaderTimeout;
    if (isLoading && !isLoaderVisible) {
      // if the request takes longer than half a second, show the loading icon
      showLoaderTimeout = setTimeout(() => {
        setIsLoaderVisible(true);
      }, 500);
    } else {
      setIsLoaderVisible(false);
    }
    return () => clearTimeout(showLoaderTimeout);
  }, [isLoading]);

  if (isLoaderVisible) {
    return <Spinner />;
  }

  if (error !== null) {
    return <div className="p-10 flex justify-center">An error occurred: {error.message}</div>;
  }

  const downloadPProf = (e: React.MouseEvent<HTMLElement>) => {
    e.preventDefault();

    const req = {
      ...profileSource.QueryRequest(),
      reportType: QueryRequest_ReportType.PPROF,
    };

    queryClient
      .query(req, {meta: metadata})
      .response.then(response => {
        if (response.report.oneofKind !== 'pprof') {
          console.log('Expected pprof report, got:', response.report.oneofKind);
          return;
        }
        const blob = new Blob([response.report.pprof], {type: 'application/octet-stream'});

        const link = document.createElement('a');
        link.href = window.URL.createObjectURL(blob);
        link.download = 'profile.pb.gz';
        link.click();
      })
      .catch(error => {
        console.error('Error while querying', error);
      });
  };

  const resetIcicleGraph = () => setCurPath([]);

  const setNewCurPath = (path: string[]) => {
    if (!arrayEquals(curPath, path)) {
      setCurPath(path);
    }
  };

  const switchProfileView = (view: string) => {
    if (navigateTo === undefined) return;

    setCurrentView(view);

    navigateTo('/', {...router, ...{currentProfileView: view}});
  };

  const sampleUnit = profileSource.ProfileType().sampleUnit;

  return (
    <>
      <div className="py-3">
        <Card>
          <Card.Body>
            <div className="flex py-3 w-full">
              <div className="w-2/5 flex space-x-4">
                <div>
                  <Button color="neutral" onClick={downloadPProf}>
                    Download pprof
                  </Button>
                </div>
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
                  className="rounded-tr-none rounded-br-none w-auto px-8 whitespace-nowrap text-ellipsis no-outline-on-buttons"
                  onClick={() => switchProfileView('table')}
                >
                  Table
                </Button>

                <Button
                  variant={`${currentView === 'both' ? 'primary' : 'neutral'}`}
                  className="rounded-tl-none rounded-tr-none rounded-bl-none rounded-br-none border-l-0 border-r-0 w-auto px-8 whitespace-nowrap no-outline-on-buttons no-outline-on-buttons text-ellipsis"
                  onClick={() => switchProfileView('both')}
                >
                  Both
                </Button>

                <Button
                  variant={`${currentView === 'icicle' ? 'primary' : 'neutral'}`}
                  className="rounded-tl-none rounded-bl-none w-auto px-8 whitespace-nowrap text-ellipsis no-outline-on-buttons"
                  onClick={() => switchProfileView('icicle')}
                >
                  Icicle Graph
                </Button>
              </div>
            </div>

            <div className="flex space-x-4 justify-between">
              {currentView === 'icicle' &&
                response !== null &&
                response.report.oneofKind === 'flamegraph' && (
                  <div className="w-full">
                    <CalcWidth throttle={300} delay={2000}>
                      <ProfileIcicleGraph
                        curPath={curPath}
                        setNewCurPath={setNewCurPath}
                        graph={response.report.flamegraph}
                        sampleUnit={sampleUnit}
                      />
                    </CalcWidth>
                  </div>
                )}

              {currentView === 'table' && (
                <div className="w-full">
                  <TopTable
                    queryClient={queryClient}
                    profileSource={profileSource}
                    sampleUnit={sampleUnit}
                  />
                </div>
              )}

              {currentView === 'both' && (
                <>
                  <div className="w-1/2">
                    <TopTable
                      queryClient={queryClient}
                      profileSource={profileSource}
                      sampleUnit={sampleUnit}
                    />
                  </div>

                  <div className="w-1/2">
                    <CalcWidth throttle={300} delay={2000}>
                      <ProfileIcicleGraph
                        curPath={curPath}
                        setNewCurPath={setNewCurPath}
                        graph={
                          response?.report.oneofKind === 'flamegraph'
                            ? response.report.flamegraph
                            : undefined
                        }
                        sampleUnit={sampleUnit}
                      />
                    </CalcWidth>
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
