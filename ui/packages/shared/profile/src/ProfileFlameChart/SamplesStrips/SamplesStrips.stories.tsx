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

// eslint-disable-next-line import/named
import {useArgs} from '@storybook/preview-api';
// eslint-disable-next-line import/named
import {Meta} from '@storybook/react';

import {NumberDuo} from '../../utils';
import {DataPoint} from './SamplesGraph';
import {SamplesStrip} from './index';

function seededRandom(seed: number): () => number {
  return () => {
    seed = (seed * 16807) % 2147483647;
    return (seed - 1) / 2147483646;
  };
}

const mockData: DataPoint[][] = [[], [], []];
const random = seededRandom(42);

for (let i = 0; i < 200; i++) {
  for (let j = 0; j < mockData.length; j++) {
    mockData[j].push({
      timestamp: 1731326092000 + i * 100,
      value: Math.floor(random() * 100),
    });
  }
}
const meta: Meta = {
  title: 'components/SamplesStrip',
  component: SamplesStrip,
};
export default meta;

export const ThreeCPUStrips = {
  args: {
    cpus: Array.from(mockData, (_, i) => ({labels: [{name: 'cpuid', value: i + 1}]})),
    data: mockData,
    selectedTimeframe: {index: 1, bounds: [mockData[0][25].timestamp, mockData[0][100].timestamp]},
    onSelectedTimeframe: (index: number, bounds: NumberDuo): void => {
      console.log('onSelectedTimeframe', index, bounds);
    },
    bounds: [mockData[0][0].timestamp, mockData[0][mockData[0].length - 1].timestamp],
    stepMs: 100,
  },
  render: function Component(args: any): JSX.Element {
    const [, setArgs] = useArgs();

    const onSelectedTimeframe = (index: number, bounds: NumberDuo): void => {
      args.onSelectedTimeframe(index, bounds);
      setArgs({...args, selectedTimeframe: {index, bounds}});
    };

    return <SamplesStrip {...args} onSelectedTimeframe={onSelectedTimeframe} />;
  },
};
