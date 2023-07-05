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

import {useEffect, useState} from 'react';

import {Provider} from 'react-redux';

import {QueryServiceClient} from '@parca/client';
import {DateTimeRange, KeyDownProvider, useParcaContext} from '@parca/components';
import {store} from '@parca/store';
import type {NavigateFunction} from '@parca/utilities';

import {ProfileSelection, ProfileSelectionFromParams, SuffixParams} from '..';
import {QuerySelection, useProfileTypes} from '../ProfileSelector';
import ProfileExplorerCompare from './ProfileExplorerCompare';
import ProfileExplorerSingle from './ProfileExplorerSingle';

interface ProfileExplorerProps {
  queryClient: QueryServiceClient;
  queryParams: any;
  navigateTo: NavigateFunction;
}

const getExpressionAsAString = (expression: string | []): string => {
  const x = Array.isArray(expression) ? expression.join() : expression;
  return x;
};

const DEFAULT_DASHBOARD_ITEMS = ['icicle'];

/* eslint-disable @typescript-eslint/naming-convention */
const sanitizeDateRange = (
  time_selection_a: string,
  from_a: number,
  to_a: number
): {time_selection_a: string; from_a: number; to_a: number} => {
  const range = DateTimeRange.fromRangeKey(time_selection_a);
  if (from_a == null && to_a == null) {
    from_a = range.getFromMs();
    to_a = range.getToMs();
  }
  return {time_selection_a: range.getRangeKey(), from_a, to_a};
};
/* eslint-enable @typescript-eslint/naming-convention */

const filterSuffix = (
  o: {[key: string]: string | string[] | undefined},
  suffix: string
): {[key: string]: string | string[] | undefined} =>
  Object.fromEntries(Object.entries(o).filter(([key]) => !key.endsWith(suffix)));

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
  const {
    loading: profileTypesLoading,
    data: profileTypesData,
    error: profileTypesError,
  } = useProfileTypes(queryClient);

  const {loader, noDataPrompt, onError} = useParcaContext();

  useEffect(() => {
    if (profileTypesError !== undefined && profileTypesError !== null) {
      onError?.(profileTypesError, 'ProfileExplorer');
    }
  }, [profileTypesError, onError]);

  /* eslint-disable @typescript-eslint/naming-convention */
  let {
    from_a,
    to_a,
    profile_name_a,
    labels_a,
    merge_from_a,
    merge_to_a,
    time_selection_a,
    compare_a,
    from_b,
    to_b,
    profile_name_b,
    labels_b,
    merge_from_b,
    merge_to_b,
    time_selection_b,
    compare_b,
    filter_by_function,
    dashboard_items,
  } = queryParams;

  // eslint-disable-next-line @typescript-eslint/naming-convention
  const expression_a = getExpressionAsAString(queryParams.expression_a);

  // eslint-disable-next-line @typescript-eslint/naming-convention
  const expression_b = getExpressionAsAString(queryParams.expression_b);

  /* eslint-enable @typescript-eslint/naming-convention */
  const [profileA, setProfileA] = useState<ProfileSelection | null>(null);
  const [profileB, setProfileB] = useState<ProfileSelection | null>(null);

  useEffect(() => {
    const mergeFrom = merge_from_a ?? undefined;
    const mergeTo = merge_to_a ?? undefined;
    const labels = typeof labels_a === 'string' ? [labels_a] : (labels_a as string[]) ?? [''];
    const profileA = ProfileSelectionFromParams(
      expression_a,
      from_a as string,
      to_a as string,
      mergeFrom as string,
      mergeTo as string,
      labels,
      filter_by_function as string
    );

    setProfileA(profileA);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [expression_a, from_a, to_a, merge_from_a, merge_to_a, labels_a, filter_by_function]);

  useEffect(() => {
    const mergeFrom = merge_from_b ?? undefined;
    const mergeTo = merge_to_b ?? undefined;
    const labels = typeof labels_b === 'string' ? [labels_b] : (labels_b as string[]) ?? [''];
    const profileB = ProfileSelectionFromParams(
      expression_b,
      from_b as string,
      to_b as string,
      mergeFrom as string,
      mergeTo as string,
      labels,
      filter_by_function as string
    );

    setProfileB(profileB);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [expression_b, from_b, to_b, merge_from_b, merge_to_b, labels_b, filter_by_function]);

  if (profileTypesLoading) {
    return <>{loader}</>;
  }

  if (profileTypesData?.types.length === 0) {
    return <>{noDataPrompt}</>;
  }

  if (profileTypesError !== undefined && profileTypesError !== null) {
    return (
      <div
        className="relative rounded border border-red-400 bg-red-100 px-4 py-3 text-red-700"
        role="alert"
      >
        <strong className="font-bold">Error! </strong>
        <span className="block sm:inline">{profileTypesError.message}</span>
      </div>
    );
  }

  const sanitizedRange = sanitizeDateRange(time_selection_a, from_a, to_a);
  time_selection_a = sanitizedRange.time_selection_a;
  from_a = sanitizedRange.from_a;
  to_a = sanitizedRange.to_a;

  if ((queryParams?.expression_a ?? '') !== '') queryParams.expression_a = expression_a;
  if ((queryParams?.expression_b ?? '') !== '') queryParams.expression_b = expression_b;

  const selectProfile = (p: ProfileSelection, suffix: string): void => {
    queryParams.expression_a = encodeURIComponent(queryParams.expression_a);
    queryParams.expression_b = encodeURIComponent(queryParams.expression_b);
    return navigateTo('/', {
      ...queryParams,
      ...SuffixParams(p.HistoryParams(), suffix),
      dashboard_items: dashboard_items ?? DEFAULT_DASHBOARD_ITEMS,
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
    profile_name: profile_name_a as string,
  };

  // Show the SingleProfileExplorer when not comparing
  if (compare_a !== 'true' && compare_b !== 'true') {
    const selectQuery = (q: QuerySelection): void => {
      const mergeParams =
        q.mergeFrom !== undefined && q.mergeTo !== undefined
          ? {merge_from_a: q.mergeFrom, merge_to_a: q.mergeTo}
          : {};
      return navigateTo(
        '/',
        // Filtering the _a suffix causes us to reset potential profile
        // selection when running a new query.
        {
          ...filterSuffix(queryParams, '_a'),
          ...{
            expression_a: encodeURIComponent(q.expression),
            from_a: q.from.toString(),
            to_a: q.to.toString(),
            time_selection_a: q.timeSelection,
            dashboard_items: dashboard_items ?? DEFAULT_DASHBOARD_ITEMS,
            ...mergeParams,
          },
        }
      );
    };

    const selectProfile = (p: ProfileSelection): void => {
      queryParams.expression_a = encodeURIComponent(queryParams.expression_a);
      return navigateTo('/', {
        ...queryParams,
        ...SuffixParams(p.HistoryParams(), '_a'),
        dashboard_items: dashboard_items ?? DEFAULT_DASHBOARD_ITEMS,
      });
    };

    const compareProfile = (): void => {
      let compareQuery = {
        compare_a: 'true',
        expression_a: encodeURIComponent(queryA.expression),
        from_a: queryA.from.toString(),
        to_a: queryA.to.toString(),
        time_selection_a: queryA.timeSelection,
        profile_name_a: queryA.profile_name,

        compare_b: 'true',
        expression_b: encodeURIComponent(queryA.expression),
        from_b: queryA.from.toString(),
        to_b: queryA.to.toString(),
        time_selection_b: queryA.timeSelection,
        profile_name_b: queryA.profile_name,
      };

      if (profileA != null) {
        compareQuery = {
          ...SuffixParams(profileA.HistoryParams(), '_a'),
          ...compareQuery,
        };
      }

      void navigateTo('/', {
        ...compareQuery,
        search_string: '',
        dashboard_items: dashboard_items ?? DEFAULT_DASHBOARD_ITEMS,
      });
    };

    return (
      <ProfileExplorerSingle
        queryClient={queryClient}
        query={queryA}
        profile={profileA}
        selectQuery={selectQuery}
        selectProfile={selectProfile}
        compareProfile={compareProfile}
        navigateTo={navigateTo}
      />
    );
  }

  const queryB = {
    expression: expression_b,
    from: parseInt(from_b as string),
    to: parseInt(to_b as string),
    timeSelection: time_selection_b as string,
    profile_name: profile_name_b as string,
  };

  const selectQueryA = (q: QuerySelection): void => {
    const mergeParams =
      q.mergeFrom !== undefined && q.mergeTo !== undefined
        ? {merge_from_a: q.mergeFrom, merge_to_a: q.mergeTo}
        : {};
    return navigateTo(
      '/',
      // Filtering the _a suffix causes us to reset potential profile
      // selection when running a new query.
      {
        ...filterSuffix(queryParams, '_a'),
        ...{
          compare_a: 'true',
          expression_a: encodeURIComponent(q.expression),
          expression_b: encodeURIComponent(expression_b),
          from_a: q.from.toString(),
          to_a: q.to.toString(),
          time_selection_a: q.timeSelection,
          filter_by_function: filter_by_function ?? '',
          dashboard_items: dashboard_items ?? DEFAULT_DASHBOARD_ITEMS,
          ...mergeParams,
        },
      }
    );
  };

  const selectQueryB = (q: QuerySelection): void => {
    const mergeParams =
      q.mergeFrom !== undefined && q.mergeTo !== undefined
        ? {merge_from_b: q.mergeFrom, merge_to_b: q.mergeTo}
        : {};
    return navigateTo(
      '/',
      // Filtering the _b suffix causes us to reset potential profile
      // selection when running a new query.
      {
        ...filterSuffix(queryParams, '_b'),
        ...{
          compare_b: 'true',
          expression_b: encodeURIComponent(q.expression),
          expression_a: encodeURIComponent(expression_a),
          from_b: q.from.toString(),
          to_b: q.to.toString(),
          time_selection_b: q.timeSelection,
          filter_by_function: filter_by_function ?? '',
          dashboard_items: dashboard_items ?? DEFAULT_DASHBOARD_ITEMS,
          ...mergeParams,
        },
      }
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
        search_string: '',
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
  const {store: reduxStore} = store();

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
