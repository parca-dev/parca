import {Query} from '@parca/parser';
import {QueryServiceClient, ProfileTypesResponse} from '@parca/client';
import {RpcError} from '@protobuf-ts/runtime-rpc';
import {ProfileSelection} from '@parca/profile';
import React, {useEffect, useMemo, useState} from 'react';
import ProfileMetricsGraph from '../ProfileMetricsGraph';
import MatchersInput from '../MatchersInput';
import MergeButton from './MergeButton';
import CompareButton from './CompareButton';
import Card from '../Card';
import {
  DateTimeRangePicker,
  DateTimeRange,
  Select,
  Button,
  ButtonGroup,
  SelectElement,
  useGrpcMetadata,
} from '../';
import {CloseIcon} from '@parca/icons';
import cx from 'classnames';

export interface QuerySelection {
  expression: string;
  from: number;
  to: number;
  merge: boolean;
  timeSelection: string;
}

interface ProfileSelectorProps {
  queryClient: QueryServiceClient;
  querySelection: QuerySelection;
  selectProfile: (source: ProfileSelection) => void;
  selectQuery: (query: QuerySelection) => void;
  closeProfile: () => void;
  enforcedProfileName: string;
  profileSelection: ProfileSelection | null;
  comparing: boolean;
  onCompareProfile: () => void;
}

interface WellKnownProfiles {
  [key: string]: {
    name: string;
    help: string;
  };
}

export interface IProfileTypesResult {
  loading: boolean;
  data?: ProfileTypesResponse;
  error?: RpcError;
}

export const useProfileTypes = (client: QueryServiceClient): IProfileTypesResult => {
  const [result, setResult] = useState<ProfileTypesResponse | undefined>(undefined);
  const [error, setError] = useState<RpcError | undefined>(undefined);
  const [loading, setLoading] = useState(true);
  const metadata = useGrpcMetadata();

  useEffect(() => {
    if (!loading) {
      return;
    }
    const call = client.profileTypes({}, {meta: metadata});
    call.response
      .then(response => setResult(response))
      .catch(error => setError(error))
      .finally(() => setLoading(false));
  }, [client, metadata, loading]);

  return {loading, data: result, error};
};

const wellKnownProfiles: WellKnownProfiles = {
  'block:contentions:count:contentions:count': {
    name: 'Block Contentions Total',
    help: 'Stack traces that led to blocking on synchronization primitives.',
  },
  'block:delay:nanoseconds:contentions:count': {
    name: 'Block Contention Time Total',
    help: 'Time delayed stack traces caused by blocking on synchronization primitives.',
  },
  // Unfortunately, fgprof does not set the period type and unit.
  'fgprof:samples:count::': {
    name: 'Fgprof Samples Total',
    help: 'CPU profile samples observed regardless of their current On/Off CPU scheduling status',
  },
  // Unfortunately, fgprof does not set the period type and unit.
  'fgprof:time:nanoseconds::': {
    name: 'Fgprof Samples Time Total',
    help: 'CPU profile measured regardless of their current On/Off CPU scheduling status in nanoseconds',
  },
  'goroutine:goroutine:count:goroutine:count': {
    name: 'Goroutine Created Total',
    help: 'Stack traces that created all current goroutines.',
  },
  'memory:alloc_objects:count:space:bytes': {
    name: 'Memory Allocated Objects Total',
    help: 'A sampling of all past memory allocations by objects.',
  },
  'memory:alloc_space:bytes:space:bytes': {
    name: 'Memory Allocated Bytes Total',
    help: 'A sampling of all past memory allocations in bytes.',
  },
  'memory:alloc_objects:count:space:bytes:delta': {
    name: 'Memory Allocated Objects Delta',
    help: 'A sampling of all memory allocations during the observation by objects.',
  },
  'memory:alloc_space:bytes:space:bytes:delta': {
    name: 'Memory Allocated Bytes Delta',
    help: 'A sampling of all memory allocations during the observation in bytes.',
  },
  'memory:inuse_objects:count:space:bytes': {
    name: 'Memory In-Use Objects',
    help: 'A sampling of memory allocations of live objects by objects.',
  },
  'memory:inuse_space:bytes:space:bytes': {
    name: 'Memory In-Use Bytes',
    help: 'A sampling of memory allocations of live objects by bytes.',
  },
  'mutex:contentions:count:contentions:count': {
    name: 'Mutex Contentions Total',
    help: 'Stack traces of holders of contended mutexes.',
  },
  'mutex:delay:nanoseconds:contentions:count': {
    name: 'Mutex Contention Time Total',
    help: 'Time delayed stack traces caused by contended mutexes.',
  },
  'process_cpu:cpu:nanoseconds:cpu:nanoseconds:delta': {
    name: 'Process CPU Nanoseconds',
    help: 'CPU profile measured by the process itself in nanoseconds.',
  },
  'process_cpu:samples:count:cpu:nanoseconds:delta': {
    name: 'Process CPU Samples',
    help: 'CPU profile samples observed by the process itself.',
  },
  'parca_agent_cpu:samples:count:cpu:nanoseconds:delta': {
    name: 'CPU Samples',
    help: 'CPU profile samples observed by Parca Agent.',
  },
};

function profileSelectElement(name: string): SelectElement {
  const wellKnown = wellKnownProfiles[name];
  if (wellKnown === undefined) return {active: <>{name}</>, expanded: <>{name}</>};

  const title = wellKnown.name.replace(/ /g, '\u00a0');
  return {
    active: <>{title}</>,
    expanded: (
      <>
        <span>{title}</span>
        <br />
        <span className="text-xs">{wellKnown.help}</span>
      </>
    ),
  };
}

const ProfileSelector = ({
  queryClient,
  querySelection,
  selectProfile,
  selectQuery,
  closeProfile,
  enforcedProfileName,
  profileSelection,
  comparing,
  onCompareProfile,
}: ProfileSelectorProps): JSX.Element => {
  const {
    loading: profileTypesLoading,
    data: profileTypesData,
    error,
  } = useProfileTypes(queryClient);
  const profileNames = useMemo(() => {
    return (error === undefined || error == null) &&
      profileTypesData !== undefined &&
      profileTypesData != null
      ? profileTypesData.types
          .map(
            type =>
              `${type.name}:${type.sampleType}:${type.sampleUnit}:${type.periodType}:${
                type.periodUnit
              }${type.delta ? ':delta' : ''}`
          )
          .sort((a: string, b: string): number => {
            return a.localeCompare(b);
          })
      : [];
  }, [profileTypesData, error]);

  const profileLabels = profileNames.map(name => ({
    key: name,
    element: profileSelectElement(name),
  }));

  const [timeRangeSelection, setTimeRangeSelection] = useState(
    DateTimeRange.fromRangeKey(querySelection.timeSelection)
  );
  const [queryExpressionString, setQueryExpressionString] = useState(querySelection.expression);

  useEffect(() => {
    if (enforcedProfileName !== '') {
      const [q, changed] = Query.parse(querySelection.expression).setProfileName(
        enforcedProfileName
      );
      if (changed) {
        setQueryExpressionString(q.toString());
        return;
      }
    }
    setQueryExpressionString(querySelection.expression);
  }, [enforcedProfileName, querySelection.expression]);

  const enforcedProfileNameQuery = (): Query => {
    const pq = Query.parse(queryExpressionString);
    const [q] = pq.setProfileName(enforcedProfileName);
    return q;
  };

  const query =
    enforcedProfileName !== '' ? enforcedProfileNameQuery() : Query.parse(queryExpressionString);
  const selectedProfileName = query.profileName();

  const setNewQueryExpression = (expr: string, merge: boolean): void => {
    selectQuery({
      expression: expr,
      from: timeRangeSelection.getFromMs(),
      to: timeRangeSelection.getToMs(),
      merge: merge,
      timeSelection: timeRangeSelection.getRangeKey(),
    });
  };

  const setQueryExpression = (): void => {
    setNewQueryExpression(query.toString(), false);
  };

  const addLabelMatcher = (key: string, value: string): void => {
    const [newQuery, changed] = Query.parse(queryExpressionString).setMatcher(key, value);
    if (changed) {
      setNewQueryExpression(newQuery.toString(), false);
    }
  };

  const setMergedSelection = (): void => {
    setNewQueryExpression(queryExpressionString, true);
  };

  const setMatchersString = (matchers: string): void => {
    const newExpressionString = `${selectedProfileName}{${matchers}}`;
    setQueryExpressionString(newExpressionString);
  };

  const setProfileName = (profileName: string): void => {
    const [newQuery, changed] = query.setProfileName(profileName);
    if (changed) {
      const q = newQuery.toString();
      setQueryExpressionString(q);
    }
  };

  const handleCompareClick = (): void => onCompareProfile();

  const searchDisabled =
    queryExpressionString === undefined ||
    queryExpressionString === '' ||
    queryExpressionString === '{}';

  const mergeDisabled = selectedProfileName === '' || querySelection.expression === undefined;
  const compareDisabled = selectedProfileName === '' || querySelection.expression === undefined;

  return (
    <Card>
      <Card.Header className={cx(comparing === true && 'overflow-x-scroll')}>
        <div className="flex space-x-4">
          {comparing && (
            <button type="button" onClick={() => closeProfile()}>
              <CloseIcon />
            </button>
          )}
          <Select
            items={profileLabels}
            selectedKey={selectedProfileName}
            onSelection={setProfileName}
            placeholder="Select profile..."
            loading={profileTypesLoading}
          />
          <MatchersInput
            queryClient={queryClient}
            setMatchersString={setMatchersString}
            runQuery={setQueryExpression}
            currentQuery={query}
          />
          <DateTimeRangePicker
            onRangeSelection={setTimeRangeSelection}
            range={timeRangeSelection}
          />
          {searchDisabled ? (
            <div>
              <Button disabled={true}>Search</Button>
            </div>
          ) : (
            <>
              <ButtonGroup style={{marginRight: 5}}>
                <MergeButton disabled={mergeDisabled} onClick={setMergedSelection} />
                {!comparing && (
                  <CompareButton disabled={compareDisabled} onClick={handleCompareClick} />
                )}
              </ButtonGroup>
              <div>
                <Button
                  onClick={(e: React.MouseEvent<HTMLElement>) => {
                    e.preventDefault();
                    setQueryExpression();
                  }}
                >
                  Search
                </Button>
              </div>
            </>
          )}
        </div>
      </Card.Header>
      {!querySelection.merge && (
        <Card.Body>
          {querySelection.expression !== undefined &&
          querySelection.expression.length > 0 &&
          querySelection.from !== undefined &&
          querySelection.to !== undefined &&
          (profileSelection == null || profileSelection.Type() !== 'merge') ? (
            <ProfileMetricsGraph
              queryClient={queryClient}
              queryExpression={querySelection.expression}
              from={querySelection.from}
              to={querySelection.to}
              select={selectProfile}
              profile={profileSelection}
              setTimeRange={(range: DateTimeRange) => {
                setTimeRangeSelection(range);
                selectQuery({
                  expression: queryExpressionString,
                  from: range.getFromMs(),
                  to: range.getToMs(),
                  merge: false,
                  timeSelection: range.getRangeKey(),
                });
              }}
              addLabelMatcher={addLabelMatcher}
            />
          ) : (
            <>
              {(profileSelection == null || profileSelection.Type() !== 'merge') && (
                <div className="my-20 text-center">
                  <p>Run a query, and the result will be displayed here.</p>
                </div>
              )}
            </>
          )}
        </Card.Body>
      )}
    </Card>
  );
};

export default ProfileSelector;
