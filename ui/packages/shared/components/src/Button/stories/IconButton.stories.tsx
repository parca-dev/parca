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

import {IconButton} from '../';

const ParcaIcon = (): JSX.Element => {
  return (
    <svg
      id="Layer_1"
      data-name="Layer 1"
      xmlns="http://www.w3.org/2000/svg"
      viewBox="70 95 102.46 138.6"
      fill="currentColor"
      height={20}
    >
      <path d="M70.27,146.13c0-30.4,21.23-50.29,50.49-50.29,30,0,51.05,21,51.05,50.1s-21.22,49.34-48.95,49.34c-18,0-32.7-8.42-39.78-22.76v60.62H70.27Zm88.54-.57c0-21.8-15.3-37.86-37.86-37.86-22.76,0-37.87,16.06-37.87,37.86S98.19,183.42,121,183.42C143.51,183.42,158.81,167.36,158.81,145.56Z" />
    </svg>
  );
};

export default {
  component: IconButton,
  title: 'Components/Button/IconButton',
};
export const Cog = {args: {icon: 'material-symbols:settings-outline-rounded', disabled: false}};
export const CustomIcon = {args: {icon: <ParcaIcon />, disabled: false}};
