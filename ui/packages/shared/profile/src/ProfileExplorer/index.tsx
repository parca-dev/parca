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

import { useEffect, useMemo } from 'react';

import {Provider} from 'react-redux';

import {QueryServiceClient} from '@parca/client';
import { KeyDownProvider, useParcaContext } from '@parca/components';
import {createStore} from '@parca/store';
import { capitalizeOnlyFirstLetter, type NavigateFunction } from '@parca/utilities';

import { useCompareModeMeta } from '../hooks/useCompareModeMeta';
import { useProfileTypes } from '../ProfileSelector';
import ProfileExplorerCompare from './ProfileExplorerCompare';
import ProfileExplorerSingle from './ProfileExplorerSingle';

interface ProfileExplorerProps {
  queryClient: QueryServiceClient;
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

const ProfileExplorerApp = ({
  queryClient,
  navigateTo,
}: ProfileExplorerProps): JSX.Element => {
  const {
    loading: profileTypesLoading,
    data: profileTypesData,
    error: profileTypesError,
  } = useProfileTypes(queryClient);

  const {loader, noDataPrompt, onError, authenticationErrorMessage} = useParcaContext();
  const { isCompareMode } = useCompareModeMeta();

  useEffect(() => {
    if (profileTypesError !== undefined && profileTypesError !== null) {
      onError?.(profileTypesError);
    }
  }, [profileTypesError, onError]);

  if (profileTypesLoading) {
    return <>{loader}</>;
  }

  if (profileTypesData?.types.length === 0) {
    return <>{noDataPrompt}</>;
  }

  if (profileTypesError !== undefined && profileTypesError !== null) {
    if (authenticationErrorMessage !== undefined && profileTypesError.code === 'UNAUTHENTICATED') {
      return <ErrorContent errorMessage={authenticationErrorMessage} />;
    }

    return <ErrorContent errorMessage={capitalizeOnlyFirstLetter(profileTypesError.message)} />;
  }

  if (isCompareMode) {
    return (
      <ProfileExplorerCompare
        queryClient={queryClient}
        navigateTo={navigateTo}
      />
    );
  }

  return (
    <ProfileExplorerSingle
      queryClient={queryClient}
      navigateTo={navigateTo}
    />
  );


};

const ProfileExplorer = ({
  queryClient,
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
          navigateTo={navigateTo}
        />
      </KeyDownProvider>
    </Provider>
  );
};

export default ProfileExplorer;
