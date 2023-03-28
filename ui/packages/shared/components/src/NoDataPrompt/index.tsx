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

import {Icon} from '@iconify/react';

declare global {
  interface Window {
    PATH_PREFIX: string;
  }
}

const pathPrefix =
  process.env.NODE_ENV === 'development'
    ? ''
    : typeof window !== 'undefined'
    ? window.PATH_PREFIX
    : '';

export const NoDataPrompt = (): JSX.Element => {
  return (
    <div className="flex justify-center">
      <div className="mt-6 flex h-96 flex-col items-center justify-center gap-6 rounded-lg bg-white px-12 text-sm shadow dark:bg-gray-700">
        <Icon icon="material-symbols:info-outline" width={40} height={40} />
        <p className="max-w-[560px] text-center">
          <span className="text-xl">The Parca server hasn&apos;t recieved any data yet!</span>{' '}
          <br />
          <br />
          Please check the{' '}
          <a href={`${pathPrefix}/targets`} className="text-blue-500">
            targets
          </a>{' '}
          page and ensure that the agents are configured correctly and sending data to the server.
          <br />
          <br />
          If you&apos;re still having trouble, please check out the{' '}
          <a
            href="https://www.parca.dev/docs/troubleshooting-parca-agent"
            className="text-blue-500"
            target="_blank"
            rel="noreferrer"
          >
            documentation
          </a>{' '}
          or join our{' '}
          <a
            href="https://discord.gg/ZgUpYgpzXy"
            className="text-blue-500"
            target="_blank"
            rel="noreferrer"
          >
            Discord
          </a>{' '}
          for help.
        </p>
      </div>
    </div>
  );
};
