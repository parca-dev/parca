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

import {useCallback} from 'react';

import {GrpcWebFetchTransport} from '@protobuf-ts/grpcweb-transport';
import {useLocation, useNavigate} from 'react-router-dom';

import {QueryServiceClient} from '@parca/client';
import {ParcaContextProvider, Spinner} from '@parca/components';
import {ProfileExplorer} from '@parca/profile';
import {selectDarkMode, useAppSelector} from '@parca/store';
import {convertToQueryParams} from '@parca/utilities';

const apiEndpoint = import.meta.env.VITE_API_ENDPOINT;

const queryClient = new QueryServiceClient(
  new GrpcWebFetchTransport({
    baseUrl: apiEndpoint === undefined ? `${window.PATH_PREFIX}/api` : `${apiEndpoint}/api`,
  })
);

const Profiles = () => {
  'use no memo';
  const navigate = useNavigate();
  // useLocation() subscribes to react-router location changes so this component
  // re-renders on navigate(). 'use no memo' ensures the re-render propagates to
  // URLStateProvider, whose no-deps effect syncs state from window.location.search.
  useLocation();
  const isDarkMode = useAppSelector(selectDarkMode);

  const navigateTo = useCallback(
    (_: string, queryParams: any, options?: {replace?: boolean}) => {
      navigate(
        {
          search: `?${convertToQueryParams(queryParams)}`,
        },
        options ?? {}
      );
    },
    [navigate]
  );

  return (
    <ParcaContextProvider
      value={{
        Spinner,
        queryServiceClient: queryClient,
        navigateTo,
        isDarkMode,
        enableSandwichView: true,
      }}
    >
      <div className="bg-white dark:bg-gray-900 p-3">
        <ProfileExplorer queryClient={queryClient} navigateTo={navigateTo} />
      </div>
    </ParcaContextProvider>
  );
};

export default Profiles;
