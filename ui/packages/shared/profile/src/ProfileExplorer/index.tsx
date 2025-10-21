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

import {useEffect, useMemo, useState} from 'react';

import {Provider} from 'react-redux';

import {QueryServiceClient} from '@parca/client';
import {DateTimeRange, KeyDownProvider, useParcaContext} from '@parca/components';
import {Query} from '@parca/parser';
import {createStore} from '@parca/store';
import {capitalizeOnlyFirstLetter, safeDecode, type NavigateFunction} from '@parca/utilities';

import {ProfileSelection, ProfileSelectionFromParams, SuffixParams} from '..';
import { QuerySelection } from '../ProfileSelector';
import {useResetFlameGraphState} from '../ProfileView/hooks/useResetFlameGraphState';
import {useResetStateOnProfileTypeChange} from '../ProfileView/hooks/useResetStateOnProfileTypeChange';
import {sumByToParam, useSumByFromParams} from '../useSumBy';
import ProfileExplorerCompare from './ProfileExplorerCompare';
import ProfileExplorerSingle from './ProfileExplorerSingle';
import { useHasProfileData } from '../useHasProfileData';

interface ProfileExplorerProps {
  queryClient: QueryServiceClient;
  queryParams: any;
  navigateTo: NavigateFunction;
}

const ErrorContent = ({errorMessage}: {errorMessage: string}): JSX.Element => {
  return (
    <div
      className="relative rounded border border-red-400 bg-red-100 px-4 py-3 text-red-700"
      role="alert"
    >
      <span className="block sm:inline">{errorMessage}</span>
    </div>
  );
};

export const getExpressionAsAString = (expression: string | []): string => {
  const x = Array.isArray(expression) ? expression.join() : expression;
  return x;
};

/* eslint-disable @typescript-eslint/naming-convention */
const sanitizeDateRange = (
  time_selection_a: string,
  from_a: number,
  to_a: number
): {time_selection_a: string; from_a: number; to_a: number} => {
  const range = DateTimeRange.fromRangeKey(time_selection_a, from_a, to_a);
  if (from_a == null && to_a == null) {
    from_a = range.getFromMs();
    to_a = range.getToMs();
  }
  return {time_selection_a: range.getRangeKey(), from_a, to_a};
};
/* eslint-enable @typescript-eslint/naming-convention */

export const filterEmptyParams = (o: Record<string, any>): Record<string, any> => {
  return Object.fromEntries(
    Object.entries(o)
      .filter(
        ([_, value]) =>
          value !== '' && value !== undefined && (Array.isArray(value) ? value.length > 0 : true)
      )
      .map(([key, value]) => {
        if (typeof value === 'string') {
          return [key, value];
        }
        if (Array.isArray(value)) {
          return [key, value];
        }
        return [key, value];
      })
  );
};

const filterSuffix = (
  o: {[key: string]: string | string[] | undefined},
  suffix: string
): {[key: string]: string | string[] | undefined} =>
  Object.fromEntries(
    Object.entries(o)
      .filter(([key]) => !key.endsWith(suffix))
      .map(([key, value]) => {
        return [key, value];
      })
  );

const swapQueryParameters = (o: {
  [key: string]: string | string[] | undefined;
}): {[key: string]: string | string[] | undefined} => {
  Object.entries(o).forEach(([key, value]) => {
    if (key.endsWith('_b')) {
      o[key.slice(0, -2) + '_a'] = value;
    }
  });
  return o;
};

const ProfileExplorerApp = ({
  queryClient,
  queryParams,
  navigateTo,
}: ProfileExplorerProps): JSX.Element => {
  const { loading: hasProfileDataLoading, data: hasProfileData, error: hasProfileDataError } = useHasProfileData(queryClient);

  const {loader, noDataPrompt, onError, authenticationErrorMessage} = useParcaContext();

  useEffect(() => {
    if (hasProfileDataError !== undefined && hasProfileDataError !== null) {
      onError?.(hasProfileDataError);
    }
  }, [hasProfileDataError, onError]);

  /* eslint-disable @typescript-eslint/naming-convention */
  let {
    from_a,
    to_a,
    merge_from_a,
    merge_to_a,
    time_selection_a,
    compare_a,
    sum_by_a,
    from_b,
    to_b,
    merge_from_b,
    merge_to_b,
    time_selection_b,
    compare_b,
    sum_by_b,
  } = queryParams;

  // eslint-disable-next-line @typescript-eslint/naming-convention
  const expression_a = getExpressionAsAString(queryParams.expression_a);

  // eslint-disable-next-line @typescript-eslint/naming-convention
  const expression_b = getExpressionAsAString(queryParams.expression_b);

  // eslint-disable-next-line @typescript-eslint/naming-convention
  const selection_a = getExpressionAsAString(queryParams.selection_a);

  // eslint-disable-next-line @typescript-eslint/naming-convention
  const selection_b = getExpressionAsAString(queryParams.selection_b);

  /* eslint-enable @typescript-eslint/naming-convention */
  const [profileA, setProfileA] = useState<ProfileSelection | null>(null);
  const [profileB, setProfileB] = useState<ProfileSelection | null>(null);

  const resetStateOnProfileTypeChange = useResetStateOnProfileTypeChange();
  const resetFlameGraphState = useResetFlameGraphState();

  const sumByA = useSumByFromParams(sum_by_a);
  const sumByB = useSumByFromParams(sum_by_b);

  useEffect(() => {
    const mergeFrom = merge_from_a ?? undefined;
    const mergeTo = merge_to_a ?? undefined;
    const profileA = ProfileSelectionFromParams(
      mergeFrom as string,
      mergeTo as string,
      selection_a
    );

    setProfileA(profileA);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [merge_from_a, merge_to_a, selection_a]);

  useEffect(() => {
    const mergeFrom = merge_from_b ?? undefined;
    const mergeTo = merge_to_b ?? undefined;
    const profileB = ProfileSelectionFromParams(
      mergeFrom as string,
      mergeTo as string,
      selection_b
    );

    setProfileB(profileB);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [merge_from_b, merge_to_b, selection_b]);

  if (hasProfileDataLoading) {
    return <>{loader}</>;
  }

  if (!hasProfileData) {
    return <>{noDataPrompt}</>;
  }

  if (hasProfileDataError !== undefined && hasProfileDataError !== null) {
    if (authenticationErrorMessage !== undefined && hasProfileDataError.code === 'UNAUTHENTICATED') {
      return <ErrorContent errorMessage={authenticationErrorMessage} />;
    }

    return <ErrorContent errorMessage={capitalizeOnlyFirstLetter(hasProfileDataError.message)} />;
  }

  const sanitizedRange = sanitizeDateRange(time_selection_a, from_a, to_a);
  time_selection_a = sanitizedRange.time_selection_a;
  from_a = sanitizedRange.from_a;
  to_a = sanitizedRange.to_a;

  if ((queryParams?.expression_a ?? '') !== '') queryParams.expression_a = safeDecode(expression_a);
  if ((queryParams?.expression_b ?? '') !== '') queryParams.expression_b = safeDecode(expression_b);

  const selectProfile = (p: ProfileSelection, suffix: string): void => {
    return navigateTo('/', {
      ...queryParams,
      ...SuffixParams(p.HistoryParams(), suffix),
    });
  };

  const selectProfileA = (p: ProfileSelection): void => {
    return selectProfile(p, '_a');
  };

  const selectProfileB = (p: ProfileSelection): void => {
    return selectProfile(p, '_b');
  };

  const queryA = {
    expression: expression_a,
    from: parseInt(from_a as string),
    to: parseInt(to_a as string),
    timeSelection: time_selection_a as string,
    sumBy: sumByA,
  };

  // Show the SingleProfileExplorer when not comparing
  if (compare_a !== 'true' && compare_b !== 'true') {
    const selectQuery = (q: QuerySelection): void => {
      const profileNameAfter = Query.parse(q.expression).profileName();
      if (profileA != null) {
        if (profileA.ProfileName() !== profileNameAfter) {
          // Reset required state when the profile type changes.
          resetStateOnProfileTypeChange();
        } else {
          // Reset the state when a new search is performed.
          resetFlameGraphState();
        }
      }

      const mergeParams =
        q.mergeFrom !== undefined && q.mergeTo !== undefined
          ? {
              merge_from_a: q.mergeFrom,
              merge_to_a: q.mergeTo,
              selection_a: q.expression,
            }
          : {};
      return navigateTo(
        '/',
        // Filtering the _a suffix causes us to reset potential profile
        // selection when running a new query.
        filterEmptyParams({
          ...filterSuffix(queryParams, '_a'),
          ...{
            expression_a: q.expression,
            from_a: q.from.toString(),
            to_a: q.to.toString(),
            time_selection_a: q.timeSelection,
            sum_by_a: sumByToParam(q.sumBy),
            ...mergeParams,
          },
        })
      );
    };

    const selectProfile = (p: ProfileSelection): void => {
      return navigateTo('/', {
        ...queryParams,
        ...SuffixParams(p.HistoryParams(), '_a'),
      });
    };

    return (
      <ProfileExplorerSingle
        queryClient={queryClient}
        query={queryA}
        profile={profileA}
        selectQuery={selectQuery}
        selectProfile={selectProfile}
        navigateTo={navigateTo}
      />
    );
  }

  const queryB = {
    expression: expression_b,
    from: parseInt(from_b as string),
    to: parseInt(to_b as string),
    timeSelection: time_selection_b as string,
    sumBy: sumByB,
  };

  const selectQueryA = (q: QuerySelection): void => {
    const mergeParams =
      q.mergeFrom !== undefined && q.mergeTo !== undefined
        ? {
            merge_from_a: q.mergeFrom,
            merge_to_a: q.mergeTo,
            selection_a: q.expression,
          }
        : {};
    return navigateTo(
      '/',
      // Filtering the _a suffix causes us to reset potential profile
      // selection when running a new query.
      filterEmptyParams({
        ...filterSuffix(queryParams, '_a'),
        ...{
          compare_a: 'true',
          expression_a: q.expression,
          expression_b,
          selection_b,
          from_a: q.from.toString(),
          to_a: q.to.toString(),
          time_selection_a: q.timeSelection,
          sum_by_a: sumByToParam(q.sumBy),
          ...mergeParams,
        },
      })
    );
  };

  const selectQueryB = (q: QuerySelection): void => {
    const mergeParams =
      q.mergeFrom !== undefined && q.mergeTo !== undefined
        ? {
            merge_from_b: q.mergeFrom,
            merge_to_b: q.mergeTo,
            selection_b: q.expression,
          }
        : {};
    return navigateTo(
      '/',
      // Filtering the _b suffix causes us to reset potential profile
      // selection when running a new query.
      filterEmptyParams({
        ...filterSuffix(queryParams, '_b'),
        ...{
          compare_b: 'true',
          expression_b: q.expression,
          expression_a,
          selection_a,
          from_b: q.from.toString(),
          to_b: q.to.toString(),
          time_selection_b: q.timeSelection,
          sum_by_b: sumByToParam(q.sumBy),
          ...mergeParams,
        },
      })
    );
  };

  const closeProfile = (card: string): void => {
    let newQueryParameters = queryParams;
    if (card === 'A') {
      newQueryParameters = swapQueryParameters(queryParams);
    }

    return navigateTo('/', {
      ...filterSuffix(newQueryParameters, '_b'),
      ...{
        compare_a: 'false',
        compare_b: 'false',
      },
    });
  };

  return (
    <ProfileExplorerCompare
      queryClient={queryClient}
      queryA={queryA}
      queryB={queryB}
      profileA={profileA}
      profileB={profileB}
      selectQueryA={selectQueryA}
      selectQueryB={selectQueryB}
      selectProfileA={selectProfileA}
      selectProfileB={selectProfileB}
      closeProfile={closeProfile}
      navigateTo={navigateTo}
    />
  );
};

const ProfileExplorer = ({
  queryClient,
  queryParams,
  navigateTo,
}: ProfileExplorerProps): JSX.Element => {
  const {additionalFlamegraphColorProfiles} = useParcaContext();

  const {store: reduxStore} = useMemo(() => {
    return createStore(additionalFlamegraphColorProfiles);
  }, [additionalFlamegraphColorProfiles]);

  return (
    <Provider store={reduxStore}>
      <KeyDownProvider>
        <ProfileExplorerApp
          queryClient={queryClient}
          queryParams={queryParams}
          navigateTo={navigateTo}
        />
      </KeyDownProvider>
    </Provider>
  );
};

export default ProfileExplorer;
