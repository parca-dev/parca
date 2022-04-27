import React, {useEffect, useState} from 'react';
import {CalcWidth} from '@parca/dynamicsize';
import {parseParams} from '@parca/functions';
import {QueryRequest, QueryResponse, QueryServiceClient, ServiceError} from '@parca/client';
import {Button, Card, useGrpcMetadata} from '@parca/components';
import * as parca_query_v1alpha1_query_pb from '@parca/client/src/parca/query/v1alpha1/query_pb';

import ProfileIcicleGraph from './ProfileIcicleGraph';
import {ProfileSource} from './ProfileSource';
import TopTable from './TopTable';

import './ProfileView.styles.css';

type NavigateFunction = (path: string, queryParams: any) => void;

interface ProfileViewProps {
  queryClient: QueryServiceClient;
  profileSource: ProfileSource;
  navigateTo?: NavigateFunction;
  compare?: boolean;
}

export interface IQueryResult {
  isLoading: boolean;
  response: QueryResponse | null;
  error: ServiceError | null;
}

function arrayEquals(a, b): boolean {
  return (
    Array.isArray(a) &&
    Array.isArray(b) &&
    a.length === b.length &&
    a.every((val, index) => val === b[index])
  );
}

export const useQuery = (
  client: QueryServiceClient,
  profileSource: ProfileSource
): IQueryResult => {
  const [result, setResult] = useState<IQueryResult>({
    isLoading: false,
    response: null,
    error: null,
  });
  const metadata = useGrpcMetadata();

  useEffect(() => {
    setResult({
      ...result,
      isLoading: true,
    });
    const req = profileSource.QueryRequest();
    req.setReportType(QueryRequest.ReportType.REPORT_TYPE_FLAMEGRAPH_UNSPECIFIED);

    client.query(
      req,
      metadata,
      (error: ServiceError | null, responseMessage: QueryResponse | null) => {
        setResult({
          isLoading: false,
          response: responseMessage,
          error: error,
        });
      }
    );
  }, [client, profileSource]);

  return result;
};

export const ProfileView = ({
  queryClient,
  profileSource,
  navigateTo,
}: ProfileViewProps): JSX.Element => {
  const router = parseParams(window.location.search);
  const currentViewFromURL = router.currentProfileView as string;
  const [curPath, setCurPath] = useState<string[]>([]);
  const [isLoaderVisible, setIsLoaderVisible] = useState<boolean>(false);
  const {isLoading, response, error} = useQuery(queryClient, profileSource);
  const [currentView, setCurrentView] = useState<string | undefined>(currentViewFromURL);
  const grpcMetadata = useGrpcMetadata();

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
    return (
      <div
        style={{
          display: 'flex',
          justifyContent: 'center',
          alignItems: 'center',
          height: 'inherit',
          marginTop: 100,
        }}
      >
        <svg
          className="animate-spin -ml-1 mr-3 h-5 w-5"
          xmlns="http://www.w3.org/2000/svg"
          fill="none"
          viewBox="0 0 24 24"
        >
          <circle
            className="opacity-25"
            cx="12"
            cy="12"
            r="10"
            stroke="currentColor"
            strokeWidth="4"
          ></circle>
          <path
            className="opacity-75"
            fill="currentColor"
            d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
          ></path>
        </svg>
        <span>Loading...</span>
      </div>
    );
  }

  if (error) {
    return <div className="p-10 flex justify-center">An error occurred: {error.message}</div>;
  }

  const downloadPProf = (e: React.MouseEvent<HTMLElement>) => {
    e.preventDefault();

    const req = profileSource.QueryRequest();
    req.setReportType(QueryRequest.ReportType.REPORT_TYPE_PPROF);

    queryClient.query(
      req,
      grpcMetadata,
      (
        error: ServiceError | null,
        responseMessage: parca_query_v1alpha1_query_pb.QueryResponse | null
      ) => {
        if (error != null) {
          console.error('Error while querying', error);
          return;
        }
        if (responseMessage !== null) {
          const bytes = responseMessage.getPprof();
          const blob = new Blob([bytes], {type: 'application/octet-stream'});

          const link = document.createElement('a');
          link.href = window.URL.createObjectURL(blob);
          link.download = 'profile.pb.gz';
          link.click();
        } else {
          console.error(error);
        }
      }
    );
  };

  const resetIcicleGraph = () => setCurPath([]);

  const setNewCurPath = (path: string[]) => {
    if (!arrayEquals(curPath, path)) {
      setCurPath(path);
    }
  };

  const switchProfileView = (view: string) => {
    if (!navigateTo) return;

    setCurrentView(view);

    navigateTo('/', {...router, ...{currentProfileView: view}});
  };

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
                  color={`${currentView === 'table' ? 'primary' : 'neutral'}`}
                  className="rounded-tr-none rounded-br-none w-auto px-8 whitespace-nowrap text-ellipsis no-outline-on-buttons"
                  onClick={() => switchProfileView('table')}
                >
                  Table
                </Button>

                <Button
                  color={`${currentView === 'both' ? 'primary' : 'neutral'}`}
                  className="rounded-tl-none rounded-tr-none rounded-bl-none rounded-br-none border-l-0 border-r-0 w-auto px-8 whitespace-nowrap no-outline-on-buttons no-outline-on-buttons text-ellipsis"
                  onClick={() => switchProfileView('both')}
                >
                  Both
                </Button>

                <Button
                  color={`${currentView === 'icicle' ? 'primary' : 'neutral'}`}
                  className="rounded-tl-none rounded-bl-none w-auto px-8 whitespace-nowrap text-ellipsis no-outline-on-buttons"
                  onClick={() => switchProfileView('icicle')}
                >
                  Icicle Graph
                </Button>
              </div>
            </div>

            <div className="flex space-x-4 justify-between">
              {currentView === 'icicle' && (
                <div className="w-full">
                  <CalcWidth throttle={300} delay={2000}>
                    <ProfileIcicleGraph
                      curPath={curPath}
                      setNewCurPath={setNewCurPath}
                      graph={response?.getFlamegraph()?.toObject()}
                    />
                  </CalcWidth>
                </div>
              )}

              {currentView === 'table' && (
                <div className="w-full">
                  <TopTable queryClient={queryClient} profileSource={profileSource} />
                </div>
              )}

              {currentView === 'both' && (
                <>
                  <div className="w-1/2">
                    <TopTable queryClient={queryClient} profileSource={profileSource} />
                  </div>

                  <div className="w-1/2">
                    <CalcWidth throttle={300} delay={2000}>
                      <ProfileIcicleGraph
                        curPath={curPath}
                        setNewCurPath={setNewCurPath}
                        graph={response?.getFlamegraph()?.toObject()}
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
