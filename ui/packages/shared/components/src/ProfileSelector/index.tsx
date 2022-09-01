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

import {Query} from '@parca/parser';
import {QueryServiceClient, ProfileTypesResponse} from '@parca/client';
import {RpcError} from '@protobuf-ts/runtime-rpc';
import {ProfileSelection} from '@parca/profile';
import React, {useEffect, useState} from 'react';
import ProfileMetricsGraph from '../ProfileMetricsGraph';
import MatchersInput from '../MatchersInput';
import MergeButton from './MergeButton';
import CompareButton from './CompareButton';
import Card from '../Card';
import {DateTimeRangePicker, DateTimeRange, Button, ButtonGroup, useGrpcMetadata} from '../';
import {CloseIcon} from '@parca/icons';
import cx from 'classnames';
import ProfileTypeSelector from '../ProfileTypeSelector';

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

  const setNewQueryExpression = (expr: string, merge: boolean): void => {
    selectQuery({
      expression: expr,
      from: timeRangeSelection.getFromMs(),
      to: timeRangeSelection.getToMs(),
      merge,
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
      <Card.Header className={cx(comparing && 'overflow-x-scroll')}>
        <div className="flex space-x-4">
          {comparing && (
            <button type="button" onClick={() => closeProfile()}>
              <CloseIcon />
            </button>
          )}
          <ProfileTypeSelector
            profileTypesData={profileTypesData}
            loading={profileTypesLoading}
            selectedKey={selectedProfileName}
            onSelection={setProfileName}
            error={error}
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
