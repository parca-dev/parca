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
import cx from 'classnames';
import {CopyToClipboard} from 'react-copy-to-clipboard';

import {Button} from '@parca/components';

interface Props {
  value: string;
  className?: string;
}

let timeoutHandle: ReturnType<typeof setTimeout> | null = null;

const ResultBox = ({value, className = ''}: Props): JSX.Element => {
  const [isCopied, setIsCopied] = useState<boolean>(false);

  const onCopy = (): void => {
    if (typeof window === 'undefined') {
      return;
    }

    setIsCopied(true);
    (window.document?.activeElement as HTMLElement)?.blur();
    if (timeoutHandle != null) {
      clearTimeout(timeoutHandle);
    }
    timeoutHandle = setTimeout(() => setIsCopied(false), 3000);
  };

  return (
    <div className={cx('flex w-full flex-row', {[className]: className?.length > 0})}>
      <span className="flex w-16 items-center justify-center rounded-l border border-r-0">
        <Icon icon="ant-design:link-outlined" />
      </span>
      <input
        type="text"
        className="w-full flex-grow border bg-inherit px-1 py-2 text-sm"
        value={value}
        readOnly
      />
      <CopyToClipboard text={value} onCopy={onCopy}>
        <Button
          variant="link"
          className="w-fit items-center whitespace-nowrap rounded-none rounded-r border border-l-0 p-4 !text-indigo-600 dark:!text-indigo-400"
        >
          {isCopied ? 'Copied!' : 'Copy Link'}
        </Button>
      </CopyToClipboard>
    </div>
  );
};

export default ResultBox;
