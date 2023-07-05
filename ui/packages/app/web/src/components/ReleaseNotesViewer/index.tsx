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

import cx from 'classnames';
import ReactMarkdown from 'react-markdown';
import {useCookie} from 'react-use';

import {Button, Modal} from '@parca/components';

interface Props {
  version: string;
}

const ReleaseNotesViewer = ({version}: Props) => {
  const [isOpen, setIsOpen] = useState<boolean>(false);
  const [isReleaseNotesAvailable, setIsReleaseNotesAvailable] = useState<boolean>(false);
  const [releaseNotes, setReleaseNotes] = useState<string>('');
  const [viewedRelease, setViewedRelease] = useCookie('viewedReleaseNotes');

  useEffect(() => {
    (async () => {
      if (version === '{{.Version}}') {
        setIsReleaseNotesAvailable(false);
        return;
      }
      const release = await fetch(
        `https://api.github.com/repos/parca-dev/parca/releases/tags/v${version}`
      ).then(res => res.json());
      setReleaseNotes(
        `Here's the list of changes in this release:\n${release.body
          .replaceAll(/(https:\/\/github.com\/([^\s]*))/g, `[$2]($1)`)
          .replaceAll(/\*\*Full.*/g, ``)
          .replaceAll(/@(\w+)/g, '[@$1](https://github.com/$1)')
          .replaceAll(/##/g, `###`) // Convert headers to one level lower
          .trim()}`
      );

      setIsReleaseNotesAvailable(true);
      if (viewedRelease !== version) {
        setIsOpen(true);
      }
    })();
  }, [version, viewedRelease]);

  const onClose = () => {
    setIsOpen(false);
    if (viewedRelease !== version) {
      setViewedRelease(version, {path: '/', expires: 180});
    }
  };

  return (
    <>
      <span
        onClick={() => {
          setIsOpen(true);
        }}
        className={cx({'cursor-pointer': isReleaseNotesAvailable})}
      >
        {version}
        {isReleaseNotesAvailable ? ` - What's new?` : ''}
      </span>
      {isReleaseNotesAvailable ? (
        <Modal
          className="h-[80vh] w-3/5"
          isOpen={isOpen}
          closeModal={onClose}
          title={`What's new in Parca ${version} 🎉`}
        >
          <div className="flex h-full flex-col pb-4 text-gray-800 dark:text-gray-200">
            <div className="prose dark:prose-invert prose-gray max-w-none flex-1 overflow-scroll pt-2">
              <ReactMarkdown
                components={{
                  // eslint-disable-next-line jsx-a11y/anchor-has-content
                  a: ({node, ...props}) => <a target="_blank" rel="noreferrer" {...props} />,
                }}
              >
                {releaseNotes}
              </ReactMarkdown>
            </div>
            <div className="mt-4 flex items-center justify-between gap-2">
              <Button variant="neutral" className="w-fit" onClick={onClose}>
                Close
              </Button>
              <Button
                className="w-fit"
                onClick={() => {
                  window.open(
                    `https://github.com/parca-dev/parca/releases/tag/v${version}`,
                    '_blank'
                  );
                  onClose();
                }}
              >
                Full Changelog
              </Button>
            </div>
          </div>
        </Modal>
      ) : null}
    </>
  );
};

export default ReleaseNotesViewer;
