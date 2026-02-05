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

import {useEffect, useRef} from 'react';

import {ProfileType} from '@parca/parser';

import {wellKnownProfiles} from '../../ProfileTypeSelector';

const CPU_PREFERRED_DIMENSIONS = ['cpu', 'cpuid', 'thread', 'thread_id'];

const GPU_DIMENSIONS = ['node', 'gpu', 'stream'];

const getWellKnownProfileName = (profileType?: ProfileType): string | undefined => {
  if (profileType == null) return undefined;
  const key = profileType.toString();
  return wellKnownProfiles[key]?.name;
};

const isOnGpuProfile = (profileType?: ProfileType): boolean => {
  const wellKnownName = getWellKnownProfileName(profileType);
  return wellKnownName === 'On-GPU';
};

const isOnCpuProfile = (profileType?: ProfileType): boolean => {
  const wellKnownName = getWellKnownProfileName(profileType);
  return wellKnownName === 'On-CPU';
};

/**
 * Auto-selects the best flamechart "group by" dimension on first load.
 *
 * For On-GPU profiles:
 *   - Selects all available from: ['node', 'gpu', 'stream']
 *
 * For On-CPU profiles:
 *   - Priority: cpu > cpuid > thread > thread_id
 *   - Adds 'node' alongside the primary dimension if available
 *
 * For all other profile types:
 *   - No auto-selection
 */
export const useAutoSelectDimension = (
  metadataLabels: string[] | undefined,
  flamechartDimension: string[] | undefined,
  setFlamechartDimension: (v: string[]) => void,
  profileType?: ProfileType
): void => {
  const hasAutoSelected = useRef(false);

  useEffect(() => {
    if (hasAutoSelected.current) return;
    if (metadataLabels == null || metadataLabels.length === 0) return;
    if ((flamechartDimension ?? []).length > 0) {
      hasAutoSelected.current = true;
      return;
    }

    if (isOnGpuProfile(profileType)) {
      const availableGpuDims = GPU_DIMENSIONS.filter(d => metadataLabels.includes(d));
      if (availableGpuDims.length > 0) {
        setFlamechartDimension(availableGpuDims.map(d => `labels.${d}`));
        hasAutoSelected.current = true;
        return;
      }
    }

    if (isOnCpuProfile(profileType)) {
      const hasNode = metadataLabels.includes('node');
      for (const name of CPU_PREFERRED_DIMENSIONS) {
        if (metadataLabels.includes(name)) {
          const dims = hasNode ? ['node', name] : [name];
          setFlamechartDimension(dims.map(d => `labels.${d}`));
          hasAutoSelected.current = true;
          return;
        }
      }
    }
  }, [metadataLabels, flamechartDimension, setFlamechartDimension, profileType]);
};
