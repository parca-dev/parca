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

import React, {useCallback, useState} from 'react';

import {Icon} from '@iconify/react';

import {QueryRequest, QueryServiceClient} from '@parca/client';
import {
  Button,
  Dropdown,
  Modal,
  useGrpcMetadata,
  useParcaContext,
  useURLState,
} from '@parca/components';

import {ProfileSource} from '../../../ProfileSource';
import {openInVSCode} from '../../../utils/vscodeDeepLink';
import {useProfileFiltersUrlState} from '../ProfileFilters/useProfileFiltersUrlState';
import ResultBox from './ResultBox';

interface Props {
  profileSource?: ProfileSource;
  queryClient?: QueryServiceClient;
  queryRequest?: QueryRequest;
  onDownloadPProf: () => void;
  pprofdownloading: boolean;
  profileViewExternalSubActions: React.ReactNode;
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
  const {Spinner} = useParcaContext();
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
              className="mt-4 h-[38px]"
              onClick={e => {
                e.preventDefault();
                void handleSubmit();
              }}
              disabled={loading || !isFormDataValid()}
              type="submit"
            >
              {loading ? <Spinner paddingClasses="p-0" /> : 'Share'}
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
  profileViewExternalSubActions,
}: Props): JSX.Element => {
  const [showProfileShareModal, setShowProfileShareModal] = useState(false);

  // Get current query state from URL for VS Code deep linking
  const [expression] = useURLState<string>('expression');
  const [timeSelection] = useURLState<string>('time_selection');
  const {appliedFilters} = useProfileFiltersUrlState();

  const handleOpenInVSCode = useCallback(() => {
    openInVSCode({
      expression: expression ?? undefined,
      timeRange: timeSelection ?? undefined,
      profileFilters: appliedFilters,
    });
  }, [expression, timeSelection, appliedFilters]);

  const actions = [
    {
      key: 'shareProfile',
      label: 'Share profile via link',
      onSelect: () => setShowProfileShareModal(true),
      id: 'h-share-profile-button',
      disabled:
        profileSource === undefined && queryClient === undefined && queryRequest === undefined,
      icon: 'material-symbols-light:link',
    },
    {
      key: 'downloadProfile',
      label: pprofdownloading != null && pprofdownloading ? 'Downloading...' : 'Download as pprof',
      onSelect: () => onDownloadPProf(),
      id: 'h-download-pprof',
      disabled: pprofdownloading,
      icon: 'material-symbols:download',
    },
    {
      key: 'openInVSCode',
      label: 'Open in VS Code',
      onSelect: handleOpenInVSCode,
      id: 'h-open-in-vscode',
      disabled: expression === undefined || expression === '',
      icon: 'simple-icons:visualstudiocode',
    },
  ];

  return (
    <>
      {profileViewExternalSubActions != null ? (
        <>
          <Button
            className="gap-2"
            variant="neutral"
            onClick={e => {
              e.preventDefault();
              onDownloadPProf();
            }}
            disabled={pprofdownloading}
            id="h-download-pprof"
          >
            {pprofdownloading != null && pprofdownloading ? 'Downloading...' : 'Download pprof'}
            <Icon icon="material-symbols:download" width={20} />
          </Button>
        </>
      ) : (
        <>
          <Dropdown
            dropdownWidth="w-48"
            element={
              <Button
                variant="neutral"
                className="flex items-center gap-2 pr-[1.7rem]"
                id="h-share-dropdown-button"
              >
                <div className="flex items-center gap-2">
                  <Icon icon="material-symbols:share" className="w-4 h-4" />

                  <span>Share</span>
                </div>
                <div className="pointer-events-none absolute inset-y-0 right-0 flex items-center pr-2 text-gray-400">
                  <Icon icon="heroicons:chevron-down-20-solid" aria-hidden="true" />
                </div>
              </Button>
            }
          >
            <span className="text-xs text-gray-400 capitalize px-2">actions</span>
            {actions.map(item => (
              <Dropdown.Item key={item.key} onSelect={item.onSelect}>
                <div id={item.id} className="flex items-center gap-2">
                  <Icon icon={item.icon} className="h-4 w-4" />
                  <span>{item.label}</span>
                </div>
              </Dropdown.Item>
            ))}
          </Dropdown>
          {profileSource !== undefined &&
            queryClient !== undefined &&
            queryRequest !== undefined && (
              <ProfileShareModal
                isOpen={showProfileShareModal}
                closeModal={() => setShowProfileShareModal(false)}
                queryRequest={queryRequest}
                queryClient={queryClient}
              />
            )}
        </>
      )}
    </>
  );
};

export default ShareButton;
