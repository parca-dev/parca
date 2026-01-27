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

import {useLocalStorage} from '@rehooks/local-storage';

const USER_PREFERENCES_KEY = 'user-preferences';

export interface UserPreferenceDetails {
  name: string;
  key: string;
  type: string;
  default: boolean | string;
  description: string;
}

export const USER_PREFERENCES: {[key: string]: UserPreferenceDetails} = {
  ENABLE_GRAPH_TRIMMING: {
    name: 'Enable trimming of nodes that are too small to be seen',
    key: 'ENABLE_GRAPH_TRIMMING',
    type: 'boolean',
    default: true,
    description:
      'When enabled, this option automatically trims nodes that are too small to be seen in the graph, making it easier to read and understand.',
  },
  FLAMEGRAPH_COLOR_PROFILE: {
    name: 'Flame graph color profile',
    key: 'FLAMEGRAPH_COLOR_PROFILE',
    type: 'string',
    default: 'ocean',
    description:
      'This setting allows you to customize the color scheme used in the Flame graph visualization.',
  },
  GRAPH_METAINFO_DOCKED: {
    name: 'Dock Graph MetaInfo',
    key: 'GRAPH_METAINFO_DOCKED',
    type: 'boolean',
    default: false,
    description:
      'When enabled, this option anchors the graph tooltip to the bottom of your screen instead of floating near your cursor.',
  },
  HIGHLIGHT_SIMILAR_STACKS: {
    name: 'Highlight similar stacks when hovering over a node',
    key: 'HIGHLIGHT_SIMILAR_STACKS',
    type: 'boolean',
    default: false,
    description:
      "When enabled, this option automatically highlights stacks that are similar to the one you're currently hovering over in the Flame graph.",
  },
  ENABLE_FLAMECHARTS: {
    name: 'Enable Flame charts',
    key: 'ENABLE_FLAMECHARTS',
    type: 'boolean',
    default: false,
    description:
      'When enabled, this option adds a new visualization type, allowing you to view your data in an Flame chart format.',
  },
  COLOR_BY: {
    name: 'Color by',
    key: 'COLOR_BY',
    type: 'string',
    default: 'binary',
    description:
      'Choose how to color the flame graph nodes. Color by binary shows different colors for different binaries, while color by filename shows different colors for different source files.',
  },
  ALIGN_FUNCTION_NAME: {
    name: 'Align function names',
    key: 'ALIGN_FUNCTION_NAME',
    type: 'string',
    default: 'left',
    description:
      'Choose how to align function names in the flame graph. Left alignment shows function names starting from the left edge, while right alignment shows them from the right edge.',
  },
  FLAMECHART_AUTO_CONFIG_POPOVER_DISMISSED: {
    name: 'Flamechart auto-configuration explanation dismissed',
    key: 'FLAMECHART_AUTO_CONFIG_POPOVER_DISMISSED',
    type: 'boolean',
    default: false,
    description:
      'When enabled, the flamechart auto-configuration explanation popover will not be shown.',
  },
} as const;

export type UserPreference = keyof typeof USER_PREFERENCES;

const useUserPreference = <T>(preferenceName: UserPreference): [T, (flag: T) => void] => {
  const [flags, setFlags] = useLocalStorage<{[flag: string]: any}>(USER_PREFERENCES_KEY, {});

  const value: T = flags[preferenceName] ?? USER_PREFERENCES[preferenceName].default;
  const setFlag = (flag: T): void => {
    setFlags({...flags, [preferenceName]: flag});
  };

  return [value, setFlag];
};

export default useUserPreference;
