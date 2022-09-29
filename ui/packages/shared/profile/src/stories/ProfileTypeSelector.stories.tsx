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

import React from 'react';

import {ComponentStory, ComponentMeta} from '@storybook/react';

import ProfileTypeSelector from '../ProfileTypeSelector';

export default {
  /* 👇 The title prop is optional.
   * See https://storybook.js.org/docs/react/configure/overview#configure-story-loading
   * to learn how to generate automatic titles
   */
  title: 'Components/ProfileTypeSelector',
  component: ProfileTypeSelector,
} as ComponentMeta<typeof ProfileTypeSelector>;

const profileTypes = {
  types: [
    {
      name: 'memory',
      sampleType: 'inuse_objects',
      sampleUnit: 'count',
      periodType: 'space',
      periodUnit: 'bytes',
      delta: false,
    },
    {
      name: 'memory',
      sampleType: 'inuse_space',
      sampleUnit: 'bytes',
      periodType: 'space',
      periodUnit: 'bytes',
      delta: false,
    },
    {
      name: 'memory',
      sampleType: 'alloc_objects',
      sampleUnit: 'count',
      periodType: 'space',
      periodUnit: 'bytes',
      delta: false,
    },
    {
      name: 'goroutine',
      sampleType: 'goroutine',
      sampleUnit: 'count',
      periodType: 'goroutine',
      periodUnit: 'count',
      delta: false,
    },
    {
      name: 'memory',
      sampleType: 'alloc_space',
      sampleUnit: 'bytes',
      periodType: 'space',
      periodUnit: 'bytes',
      delta: false,
    },
    {
      name: 'process_cpu',
      sampleType: 'cpu',
      sampleUnit: 'nanoseconds',
      periodType: 'cpu',
      periodUnit: 'nanoseconds',
      delta: true,
    },
    {
      name: 'process_cpu',
      sampleType: 'samples',
      sampleUnit: 'count',
      periodType: 'cpu',
      periodUnit: 'nanoseconds',
      delta: true,
    },
  ],
};

const onSelection = () => {};

//👇 We create a “template” of how args map to rendering
// const Template: ComponentStory<typeof ProfileView> = args => <ProfileView {...args} />;

//👇 Each story then reuses that template
// export const Primary = Template.bind({});
// Primary.args = {sampleUnit: 'count', label: 'Button'};

export const Primary: ComponentStory<typeof ProfileTypeSelector> = () => (
  <ProfileTypeSelector
    profileTypesData={profileTypes}
    onSelection={onSelection}
    selectedKey="process_cpu:samples:count:cpu:nanoseconds:delta"
    error={undefined}
  />
);
