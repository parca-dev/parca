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

<<<<<<< HEAD
// eslint-disable-next-line import/named
import { useArgs } from '@storybook/preview-api';
// eslint-disable-next-line import/named
import { Meta } from '@storybook/react';
=======
import {useArgs} from '@storybook/preview-api';
import {Meta} from '@storybook/react';
>>>>>>> origin/metrics-graph-strips

import { DataPoint, NumberDuo } from './AreaGraph';
import { MetricsGraphStrips } from './index';

const mockData: DataPoint[][] = [[], [], []];

for (let i = 0; i < 200; i++) {
  for (let j = 0; j < mockData.length; j++) {
    mockData[j].push({
      timestamp: 1731326092000 + i * 100,
      value: Math.floor(Math.random() * 100),
    });
  }
}
const meta: Meta = {
  title: 'components/MetricsGraphStrips',
  component: MetricsGraphStrips,
};
export default meta;

export const ThreeCPUStrips = {
  args: {
    cpus: Array.from(mockData, (_, i) => `CPU ${i + 1}`),
    data: mockData,
    selectedTimeline: { index: 1, bounds: [mockData[0][25].timestamp, mockData[0][100].timestamp] },
    onSelectedTimeline: (index: number, bounds: NumberDuo): void => {
      console.log('onSelectedTimeline', index, bounds);
    },
  },
  render: function Component(args: any): JSX.Element {
    const [, setArgs] = useArgs();

    const onSelectedTimeline = (index: number, bounds: NumberDuo): void => {
      args.onSelectedTimeline(index, bounds);
      setArgs({ ...args, selectedTimeline: { index, bounds } });
    };

    return <MetricsGraphStrips {...args} onSelectedTimeline={onSelectedTimeline} />;
  },
};
