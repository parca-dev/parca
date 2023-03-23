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

import Pill from '..';

export default {
  component: Pill,
  title: 'Components/Pill',
};

export const Primary = {args: {variant: 'primary', children: <>Primary Pill</>}};
export const Success = {args: {variant: 'success', children: <>Success Pill</>}};
export const Danger = {args: {variant: 'danger', children: <>Danger Pill</>}};
export const Warning = {args: {variant: 'warning', children: <>Warning Pill</>}};
export const Info = {args: {variant: 'info', children: <>Info Pill</>}};
export const Neutral = {args: {variant: 'neutral', children: <>Neutral Pill</>}};
