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
import {Button, Dropdown, Modal, useGrpcMetadata} from '@parca/components';

import {ProfileSource} from '../../ProfileSource';
import ResultBox from './ResultBox';

interface Props {
  profileSource?: ProfileSource;
  queryClient?: QueryServiceClient;
  queryRequest?: QueryRequest;
  onDownloadPProf: () => void;
  pprofdownloading: boolean;
}

interface ProfileShareModalProps {
  queryRequest: QueryRequest;
  queryClient: QueryServiceClient;
  isOpen: boolean;
  closeModal: () => void;
}

const ProfileShareModal = ({
  isOpen,
  closeModal,
  queryRequest,
  queryClient,
}: ProfileShareModalProps): JSX.Element => {
  const [isShared, setIsShared] = useState(false);
  const [loading, setLoading] = useState<boolean>(false);
  const [error, setError] = useState<string>('');
  const [description, setDescription] = useState<string>('');
  const [sharedLink, setSharedLink] = useState<string>('');
  const metadata = useGrpcMetadata();
  const isFormDataValid = (): boolean => true;

  const handleSubmit: () => Promise<void> = async () => {
    try {
      setLoading(true);
      const {response} = await queryClient.shareProfile(
        {queryRequest, description},
        {meta: metadata}
      );
      setSharedLink(response.link);
      setLoading(false);
      setIsShared(true);
    } catch (err) {
      if (err instanceof Error) {
        console.error(err);
        setLoading(false);
        // https://github.com/microsoft/TypeScript/issues/38347
        // eslint-disable-next-line @typescript-eslint/no-base-to-string
        setError(err.toString());
      }
    }
  };

  const onClose = (): void => {
    setLoading(false);
    setError('');
    setDescription('');
    setIsShared(false);
    closeModal();
  };

  return (
    <Modal isOpen={isOpen} closeModal={onClose} title="Share Profile" className="w-[420px]">
      <div className="py-2">
        <p className="text-sm text-gray-500 dark:text-gray-300">
          Note: Shared profiles can be accessed by anyone with the link, even from people outside
          your organisation.
        </p>
        {!isShared || error?.length > 0 ? (
          <>
            <p className="mb-2 mt-3 text-sm text-gray-500 dark:text-gray-300">
              Enter a description (optional)
            </p>
            <textarea
              className="w-full border bg-inherit px-2 py-2 text-sm text-gray-500 dark:text-gray-300"
              value={description}
              onChange={e => setDescription(e.target.value)}
            ></textarea>
            <Button
              variant="primary"
              className="mt-4"
              onClick={e => {
                e.preventDefault();
                void handleSubmit();
              }}
              disabled={loading || !isFormDataValid()}
              type="submit"
            >
              {loading ? 'Sharing' : 'Share'}
            </Button>
            {error !== '' ? <p>Something went wrong please try again</p> : null}
          </>
        ) : (
          <>
            <ResultBox value={sharedLink} className="mt-4" />
            <div className="mt-8 flex justify-center">
              <Button variant="neutral" className="w-fit" onClick={onClose}>
                Close
              </Button>
            </div>
          </>
        )}
      </div>
    </Modal>
  );
};

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
