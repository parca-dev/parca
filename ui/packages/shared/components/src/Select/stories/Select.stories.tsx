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

import Select from '..';

export default {
  component: Select,
  title: 'Components/Select ',
};

const items = [
  {
    key: {key: 'Block Contentions Total'},
    element: {
      active: <>Block Contentions Total</>,
      expanded: (
        <>
          <span>Block Contentions Total</span>
          <br />
          <span className="text-xs">
            Stack traces that led to blocking on synchronization primitives.
          </span>
        </>
      ),
    },
  },
  {
    key: {key: 'Goroutine Created Total'},
    element: {
      active: <>Goroutine Created Total</>,
      expanded: (
        <>
          <span>Goroutine Created Total</span>
          <br />
          <span className="text-xs">Stack traces that created all current goroutines.</span>
        </>
      ),
    },
  },
  {
    key: {key: 'Memory Allocated Bytes Total'},
    element: {
      active: <>Memory Allocated Bytes Total</>,
      expanded: (
        <>
          <span>Memory Allocated Bytes Total</span>
          <br />
          <span className="text-xs">A sampling of all past memory allocations in bytes.</span>
        </>
      ),
    },
  },
  {
    key: {key: 'Memory Allocated Bytes Delta'},
    element: {
      active: <>Memory Allocated Bytes Delta</>,
      expanded: (
        <>
          <span>Memory Allocated Bytes Delta</span>
          <br />
          <span className="text-xs">
            A sampling of all memory allocations during the observation in bytes.
          </span>
        </>
      ),
    },
  },
  {
    key: {key: 'Process CPU Nanoseconds'},
    element: {
      active: <>Process CPU Nanoseconds</>,
      expanded: (
        <>
          <span>Process CPU Nanoseconds</span>
          <br />
          <span className="text-xs">
            CPU profile measured by the process itself in nanoseconds.
          </span>
        </>
      ),
    },
  },
  {
    key: {key: 'Process CPU Samples'},
    element: {
      active: <>Process CPU Samples</>,
      expanded: (
        <>
          <span>Process CPU Samples</span>
          <br />
          <span className="text-xs">CPU profile samples observed by the process itself.</span>
        </>
      ),
    },
  },
];

export const Default = {
  args: {
    placeholder: 'Select Profile',
    items,
    selectedKey: 'Block Contentions Total',
  },
};

export const Loading = {
  args: {
    placeholder: 'Select Profile',
    items,
    loading: true,
    selectedKey: 'Block Contentions Total',
  },
};
