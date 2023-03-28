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

import Modal from '../';
import {Button} from '../../Button';

const StateWrappedComponent = (): JSX.Element => {
  const [isOpen, setIsOpen] = useState<boolean>(false);
  const [description, setDescription] = useState<string>('');

  return (
    <>
      <Button color="neutral" onClick={() => setIsOpen(true)}>
        Open Modal
      </Button>

      <Modal
        isOpen={isOpen}
        closeModal={() => setIsOpen(false)}
        title="Share Profile"
        className="w-[420px]"
      >
        <form className="py-2">
          <p className="text-sm text-gray-500 dark:text-gray-300">
            Note: Shared profiles can be accessed by anyone with the link, even from people outside
            your organisation.
          </p>
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
              }}
              type="submit"
            >
              Share
            </Button>
          </>
        </form>
      </Modal>
    </>
  );
};

export default StateWrappedComponent;
