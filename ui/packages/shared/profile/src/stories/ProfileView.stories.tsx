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

import {ProfileView} from '../ProfileView';
import flamegraphData from './mockdata/flamegraphData.json';

export default {
  /* 👇 The title prop is optional.
   * See https://storybook.js.org/docs/react/configure/overview#configure-story-loading
   * to learn how to generate automatic titles
   */
  title: 'Components/ProfileView',
  component: ProfileView,
} as ComponentMeta<typeof ProfileView>;

const mockVisState = {
  currentView: 'icicle',
  setCurrentView: () => {},
};

const downloadPprof = () => {};

//👇 We create a “template” of how args map to rendering
// const Template: ComponentStory<typeof ProfileView> = args => <ProfileView {...args} />;

//👇 Each story then reuses that template
// export const Primary = Template.bind({});
// Primary.args = {sampleUnit: 'count', label: 'Button'};

export const Primary: ComponentStory<typeof ProfileView> = () => (
  <ProfileView
    flamegraphData={flamegraphData}
    sampleUnit="count"
    profileVisState={mockVisState}
    onDownloadPProf={downloadPprof}
  />
);
