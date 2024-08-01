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

import {useState} from 'react';

import {Icon} from '@iconify/react';

import {QueryRequest, QueryServiceClient} from '@parca/client';
import {Button, Dropdown} from '@parca/components';

import {ProfileSource} from '../ProfileSource';
import {ProfileShareModal} from '../components/ProfileShareButton';

interface Props {
  profileSource?: ProfileSource;
  queryClient?: QueryServiceClient;
  queryRequest?: QueryRequest;
  onDownloadPProf: () => void;
  pprofdownloading: boolean;
}

const ShareButton = ({
  queryRequest,
  queryClient,
  profileSource,
  onDownloadPProf,
  pprofdownloading,
}: Props): JSX.Element => {
  const [showProfileShareModal, setShowProfileShareModal] = useState(false);

  const actions = [
    {
      key: 'shareProfile',
      label: 'Share profile',
      onSelect: () => setShowProfileShareModal(true),
      id: 'h-share-profile-button',
      disabled:
        profileSource === undefined && queryClient === undefined && queryRequest === undefined,
    },
    {
      key: 'downloadProfile',
      label: pprofdownloading != null && pprofdownloading ? 'Downloading...' : 'Download pprof',
      onSelect: () => onDownloadPProf(),
      id: 'h-download-pprof',
      disabled: pprofdownloading,
    },
  ];

  return (
    <>
      <Dropdown
        element={
          <Button variant="neutral">
            Share
            <Icon icon="material-symbols:share" className="h-5 w-5 ml-2" />
          </Button>
        }
      >
        <span className="text-xs text-gray-400 capitalize px-2">actions</span>
        {actions.map(item => (
          <Dropdown.Item key={item.key} onSelect={item.onSelect}>
            <div id={item.id} className="flex items-center">
              {item.label}
            </div>
          </Dropdown.Item>
        ))}
      </Dropdown>
      {profileSource !== undefined && queryClient !== undefined && queryRequest !== undefined && (
        <ProfileShareModal
          isOpen={showProfileShareModal}
          closeModal={() => setShowProfileShareModal(false)}
          queryRequest={queryRequest}
          queryClient={queryClient}
        />
      )}
    </>
  );
};

export default ShareButton;
