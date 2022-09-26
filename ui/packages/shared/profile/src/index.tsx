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
import type {Props as CallgraphProps} from '@parca/profile/src/Callgraph';
import ProfileExplorer from './ProfileExplorer';
import ProfileTypeSelector from './ProfileTypeSelector';

export * from './IcicleGraph';
export * from './ProfileIcicleGraph';
export * from './ProfileSource';
export * from './ProfileView';
export * from './ProfileViewWithData';
export * from './utils';

export type {CallgraphProps};

const Callgraph = React.lazy(async () => await import('@parca/profile/src/Callgraph'));

export {Callgraph, ProfileExplorer, ProfileTypeSelector};
