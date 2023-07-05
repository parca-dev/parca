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
import {Button, Modal, useGrpcMetadata} from '@parca/components';

import ResultBox from './ResultBox';

interface Props {
  queryRequest: QueryRequest;
  queryClient: QueryServiceClient;
  disabled?: boolean;
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
      <form className="py-2">
        <p className="text-sm text-gray-500 dark:text-gray-300">
          Note: Shared profiles can be accessed by anyone with the link, even from people outside
          your organisation.
        </p>
        {!isShared || error?.length > 0 ? (
          <>
            <p className="mt-3 mb-2 text-sm text-gray-500 dark:text-gray-300">
              Enter a description (optional)
            </p>
            <textarea
              className="w-full border bg-inherit px-2 py-2 text-sm text-gray-500 dark:text-gray-300"
              value={description}
              onChange={e => setDescription(e.target.value)}
            ></textarea>
            <Button
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
      </form>
    </Modal>
  );
};

const ProfileShareButton = ({queryRequest, queryClient, disabled = false}: Props): JSX.Element => {
  const [isOpen, setIsOpen] = useState<boolean>(false);

  return (
    <>
      <Button color="neutral" onClick={() => setIsOpen(true)} disabled={disabled}>
        <Icon icon="ei:share-apple" width={20} />
      </Button>
      <ProfileShareModal
        isOpen={isOpen}
        closeModal={() => setIsOpen(false)}
        queryRequest={queryRequest}
        queryClient={queryClient}
      />
    </>
  );
};

export default ProfileShareButton;
