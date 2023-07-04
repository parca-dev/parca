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

import React, {useEffect, useState} from 'react';

import {RpcError} from '@protobuf-ts/runtime-rpc';

import {ProfileTypesResponse, QueryServiceClient} from '@parca/client';
import {
  Button,
  ButtonGroup,
  Card,
  DateTimeRange,
  DateTimeRangePicker,
  IconButton,
  useGrpcMetadata,
} from '@parca/components';
import {CloseIcon} from '@parca/icons';
import {Query} from '@parca/parser';
import {getStepDuration, getStepDurationInMilliseconds} from '@parca/utilities';

import {MergedProfileSelection, ProfileSelection} from '..';
import MatchersInput from '../MatchersInput/index';
import ProfileMetricsGraph from '../ProfileMetricsGraph';
import ProfileTypeSelector from '../ProfileTypeSelector/index';
import CompareButton from './CompareButton';

export interface QuerySelection {
  expression: string;
  from: number;
  to: number;
  timeSelection: string;
  mergeFrom?: number;
  mergeTo?: number;
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

  const setNewQueryExpression = (expr: string): void => {
    const query = enforcedProfileName !== '' ? enforcedProfileNameQuery() : Query.parse(expr);
    const delta = query.profileType().delta;
    const from = timeRangeSelection.getFromMs();
    const to = timeRangeSelection.getToMs();
    const mergeParams = delta
      ? {
          mergeFrom: from,
          mergeTo: to,
        }
      : {};

    selectQuery({
      expression: expr,
      from,
      to,
      timeSelection: timeRangeSelection.getRangeKey(),
      ...mergeParams,
    });
  };

  const setQueryExpression = (): void => {
    setNewQueryExpression(query.toString());
  };

  const addLabelMatcher = (key: string, value: string): void => {
    // When a user clicks on a label on the metrics graph tooltip,
    // replace single `\` in the `value` string with doubles `\\` if available.
    const newValue = value.includes('\\') ? value.replaceAll('\\', '\\\\') : value;
    const [newQuery, changed] = Query.parse(queryExpressionString).setMatcher(key, newValue);
    if (changed) {
      // TODO: Change this to store the query object
      setNewQueryExpression(newQuery.toString());
    }
  };

  const setMatchersString = (matchers: string): void => {
    const newExpressionString = `${selectedProfileName}{${matchers}}`;
    setQueryExpressionString(newExpressionString);
  };

  const setProfileName = (profileName: string | undefined): void => {
    if (profileName === undefined) {
      return;
    }
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

  const compareDisabled = selectedProfileName === '' || querySelection.expression === undefined;

  return (
    <Card className="overflow-visible">
      <Card.Header className="flex !items-center space-x-2">
        <div className="flex w-full flex-wrap items-center justify-start space-x-2 space-y-1">
          <div className="ml-2 mt-1">
            <ProfileTypeSelector
              profileTypesData={profileTypesData}
              loading={profileTypesLoading}
              selectedKey={selectedProfileName}
              onSelection={setProfileName}
              error={error}
            />
          </div>
          <div className="w-full flex-1">
            <MatchersInput
              queryClient={queryClient}
              setMatchersString={setMatchersString}
              runQuery={setQueryExpression}
              currentQuery={query}
            />
          </div>
          <DateTimeRangePicker
            onRangeSelection={setTimeRangeSelection}
            range={timeRangeSelection}
          />
          <ButtonGroup>
            {!searchDisabled && (
              <>
                {!comparing && (
                  <CompareButton disabled={compareDisabled} onClick={handleCompareClick} />
                )}
              </>
            )}
            <Button
              disabled={searchDisabled}
              onClick={(e: React.MouseEvent<HTMLElement>) => {
                e.preventDefault();
                setQueryExpression();
              }}
            >
              Search
            </Button>
          </ButtonGroup>
        </div>
        <div>{comparing && <IconButton onClick={() => closeProfile()} icon={<CloseIcon />} />}</div>
      </Card.Header>
      {
        <Card.Body>
          {querySelection.expression !== undefined &&
          querySelection.expression.length > 0 &&
          querySelection.from !== undefined &&
          querySelection.to !== undefined ? (
            <ProfileMetricsGraph
              queryClient={queryClient}
              queryExpression={querySelection.expression}
              from={querySelection.from}
              to={querySelection.to}
              profile={profileSelection}
              setTimeRange={(range: DateTimeRange) => {
                const from = range.getFromMs();
                const to = range.getToMs();
                let mergedProfileParams = {};
                if (query.profileType().delta) {
                  mergedProfileParams = {mergeFrom: from, mergeTo: to};
                }
                setTimeRangeSelection(range);
                selectQuery({
                  expression: queryExpressionString,
                  from,
                  to,
                  timeSelection: range.getRangeKey(),
                  ...mergedProfileParams,
                });
              }}
              addLabelMatcher={addLabelMatcher}
              onPointClick={(timestamp, labels, queryExpression) => {
                // TODO: Pass the query object via click rather than queryExpression
                let query = Query.parse(queryExpression);
                labels.forEach(l => {
                  const [newQuery, updated] = query.setMatcher(l.name, l.value);
                  if (updated) {
                    query = newQuery;
                  }
                });

                const stepDuration = getStepDuration(querySelection.from, querySelection.to);
                const stepDurationInMilliseconds = getStepDurationInMilliseconds(stepDuration);
                const mergeFrom = timestamp;
                const mergeTo = query.profileType().delta
                  ? mergeFrom + stepDurationInMilliseconds
                  : mergeFrom;
                selectProfile(new MergedProfileSelection(mergeFrom, mergeTo, query));
              }}
            />
          ) : (
            <>
              {profileSelection == null && (
                <div className="my-20 text-center">
                  <p>Run a query, and the result will be displayed here.</p>
                </div>
              )}
            </>
          )}
        </Card.Body>
      }
    </Card>
  );
};

export default ProfileSelector;
