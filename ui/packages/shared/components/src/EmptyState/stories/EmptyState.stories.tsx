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

import EmptyState from '..';

const ATTENTION_ICON = (
  <svg width="32" height="32" viewBox="0 0 48 48">
    <mask id="ipSAttention0">
      <g fill="none">
        <path
          fill="#fff"
          stroke="#fff"
          strokeLinejoin="round"
          strokeWidth="4"
          d="M24 44a19.937 19.937 0 0 0 14.142-5.858A19.937 19.937 0 0 0 44 24a19.938 19.938 0 0 0-5.858-14.142A19.937 19.937 0 0 0 24 4A19.938 19.938 0 0 0 9.858 9.858A19.938 19.938 0 0 0 4 24a19.937 19.937 0 0 0 5.858 14.142A19.938 19.938 0 0 0 24 44Z"
        />
        <path
          fill="#000"
          fillRule="evenodd"
          d="M24 37a2.5 2.5 0 1 0 0-5a2.5 2.5 0 0 0 0 5Z"
          clipRule="evenodd"
        />
        <path
          stroke="#000"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth="4"
          d="M24 12v16"
        />
      </g>
    </mask>
    <path fill="currentColor" d="M0 0h48v48H0z" mask="url(#ipSAttention0)" />
  </svg>
);

export default {
  component: EmptyState,
  title: 'Components/EmptyState ',
};

export const Default = {
  args: {
    title: "Oops! You're not sending us any data yet!",
    isEmpty: true,
    body: (
      <>
        <p>
          For additional information see the{' '}
          <a
            className="text-blue-500"
            href="https://www.parca.dev/docs/parca-agent-design#target-discovery"
          >
            Target Discovery
          </a>{' '}
          documentation
        </p>
      </>
    ),
  },
};

export const WithIcon = {
  args: {
    icon: ATTENTION_ICON,
    isEmpty: true,
    body: (
      <>
        <p>
          For additional information see the{' '}
          <a
            className="text-blue-500"
            href="https://www.parca.dev/docs/parca-agent-design#target-discovery"
          >
            Target Discovery
          </a>{' '}
          documentation
        </p>
      </>
    ),
  },
};
