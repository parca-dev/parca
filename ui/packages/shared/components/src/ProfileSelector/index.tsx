import {QueryServiceClient, ServiceError, ValuesRequest, ValuesResponse} from '@parca/client';
import {Query} from '@parca/parser';
import {ProfileSelection} from '@parca/profile';
import React, {useEffect, useState} from 'react';
import ProfileMetricsGraph from '../ProfileMetricsGraph';
import MatchersInput from '../MatchersInput';
import MergeButton from './MergeButton';
import CompareButton from './CompareButton';
import Card from '../Card';
import {DateTimeRangePicker, DateTimeRange, Select, Button, ButtonGroup, SelectElement} from '../';
import {CloseIcon} from '@parca/icons';

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

export interface ILabelValuesResult {
  response: ValuesResponse.AsObject | null;
  error: ServiceError | null;
}

interface WellKnownProfiles {
  [key: string]: {
    name: string;
    help: string;
  };
}

export const useLabelValues = (
  client: QueryServiceClient,
  labelName: string
): ILabelValuesResult => {
  const [result, setResult] = useState<ILabelValuesResult>({
    response: null,
    error: null,
  });

  useEffect(() => {
    const req = new ValuesRequest();
    req.setLabelName(labelName);

    client.values(req, (error: ServiceError | null, responseMessage: ValuesResponse | null) => {
      const res = responseMessage == null ? null : responseMessage.toObject();

      setResult({
        response: res,
        error: error,
      });
    });
  }, [client, labelName]);

  return result;
};

const wellKnownProfiles: WellKnownProfiles = {
  block_total_contentions_count: {
    name: 'Block Contentions Total',
    help: 'Stack traces that led to blocking on synchronization primitives.',
  },
  block_total_delay_nanoseconds: {
    name: 'Block Contention Time Total',
    help: 'Time delayed stack traces caused by blocking on synchronization primitives.',
  },
  fgprof_samples_count: {
    name: 'Fgprof Samples Total',
    help: 'CPU profile samples observed regardless of their current On/Off CPU scheduling status',
  },
  fgprof_time_nanoseconds: {
    name: 'Fgprof Samples Time Total',
    help: 'CPU profile measured regardless of their current On/Off CPU scheduling status in nanoseconds',
  },
  goroutine_total_goroutine_count: {
    name: 'Goroutine Created Total',
    help: 'Stack traces of all current goroutines.',
  },
  memory_total_alloc_objects_count: {
    name: 'Memory Allocated Objects Total',
    help: 'A sampling of all past memory allocations by objects.',
  },
  memory_total_alloc_space_bytes: {
    name: 'Memory Allocated Bytes Total',
    help: 'A sampling of all past memory allocations in bytes.',
  },
  memory_total_inuse_objects_count: {
    name: 'Memory In-Use Objects',
    help: 'A sampling of memory allocations of live objects by objects.',
  },
  memory_total_inuse_space_bytes: {
    name: 'Memory In-Use Bytes',
    help: 'A sampling of memory allocations of live objects by bytes.',
  },
  mutex_total_contentions_count: {
    name: 'Mutex Contentions Total',
    help: 'Stack traces of holders of contended mutexes.',
  },
  mutex_total_delay_nanoseconds: {
    name: 'Mutex Contention Time Total',
    help: 'Time delayed stack traces caused by contended mutexes.',
  },
  process_cpu_cpu_nanoseconds: {
    name: 'Process CPU Nanoseconds',
    help: 'CPU profile measured by the process itself in nanoseconds.',
  },
  process_cpu_samples_count: {
    name: 'Process CPU Samples',
    help: 'CPU profile samples observed by the process itself.',
  },
  parca_agent_cpu_samples_count: {
    name: 'CPU Samples',
    help: 'CPU profile samples observed by Parca Agent.',
  },
  threadcreate_total_threadcreate_count: {
    name: 'Threads Created Total',
    help: 'Stack traces that led to the creation of new OS threads.',
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
  const {response, error} = useLabelValues(queryClient, '__name__');
  const profileNames =
    (error === undefined || error == null) && response !== undefined && response != null
      ? response.labelValuesList
      : [];

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
      setQueryExpressionString(newQuery.toString());
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
