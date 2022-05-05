import React, {useState, useEffect} from 'react';
import MetricsGraph from '../MetricsGraph';
import {ProfileSelection, SingleProfileSelection} from '@parca/profile';
import {QueryServiceClient, QueryRangeResponse, Label, Timestamp} from '@parca/client';
import {RpcError} from '@protobuf-ts/runtime-rpc';
import {DateTimeRange, Spinner, useGrpcMetadata} from '../';
import {Query} from '@parca/parser';

interface ProfileMetricsGraphProps {
  queryClient: QueryServiceClient;
  queryExpression: string;
  profile: ProfileSelection | null;
  from: number;
  to: number;
  select: (source: ProfileSelection) => void;
  setTimeRange: (range: DateTimeRange) => void;
  addLabelMatcher: (key: string, value: string) => void;
}

export interface IQueryRangeResult {
  response: QueryRangeResponse | null;
  isLoading: boolean;
  error: RpcError | null;
}

export const useQueryRange = (
  client: QueryServiceClient,
  queryExpression: string,
  start: number,
  end: number
): IQueryRangeResult => {
  const [result, setResult] = useState<IQueryRangeResult>({
    response: null,
    isLoading: false,
    error: null,
  });
  const metadata = useGrpcMetadata();

  useEffect(() => {
    setResult({
      ...result,
      isLoading: true,
    });

    const call = client.queryRange(
      {
        query: queryExpression,
        start: Timestamp.fromDate(new Date(start)),
        end: Timestamp.fromDate(new Date(end)),
        limit: 0,
      },
      {meta: metadata}
    );

    call.response
      .then(response => setResult({response: response, isLoading: false, error: null}))
      .catch(error => setResult({error: error, isLoading: false, response: null}));
  }, [client, queryExpression, start, end, metadata]);

  return result;
};

const ProfileMetricsGraph = ({
  queryClient,
  queryExpression,
  profile,
  from,
  to,
  select,
  setTimeRange,
  addLabelMatcher,
}: ProfileMetricsGraphProps): JSX.Element => {
  const {isLoading, response, error} = useQueryRange(queryClient, queryExpression, from, to);
  const [isLoaderVisible, setIsLoaderVisible] = useState<boolean>(false);

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
    return (
      <div
        className="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded relative"
        role="alert"
      >
        <strong className="font-bold">Error! </strong>
        <span className="block sm:inline">{error.message}</span>
      </div>
    );
  }

  const series = response?.series;
  if (series !== null && series !== undefined && series?.length > 0) {
    const handleSampleClick = (timestamp: number, value: number, labels: Label[]): void => {
      select(
        new SingleProfileSelection(Query.parse(queryExpression).profileName(), labels, timestamp)
      );
    };

    return (
      <div
        className="dark:bg-gray-700 rounded border-gray-300 dark:border-gray-500"
        style={{borderWidth: 1}}
      >
        <MetricsGraph
          data={series}
          from={from}
          to={to}
          profile={profile as SingleProfileSelection}
          setTimeRange={setTimeRange}
          onSampleClick={handleSampleClick}
          onLabelClick={addLabelMatcher}
          width={0}
          sampleUnit={Query.parse(queryExpression).profileType().sampleUnit}
        />
      </div>
    );
  }
  return (
    <div className="grid grid-cols-1">
      <div className="py-20 flex justify-center">
        <p className="m-0">No data found. Try a different query.</p>
      </div>
    </div>
  );
};

export default ProfileMetricsGraph;
