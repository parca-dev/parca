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

import {useMemo} from 'react';

import {RpcError} from '@protobuf-ts/runtime-rpc';

import {ProfileType, ProfileTypesResponse} from '@parca/client';
import {Select, type SelectElement} from '@parca/components';

interface WellKnownProfile {
  name: string;
  help: string;
}

interface WellKnownProfiles {
  [key: string]: WellKnownProfile;
}

export const wellKnownProfiles: WellKnownProfiles = {
  'block:contentions:count:contentions:count': {
    name: 'Block Contentions Total',
    help: 'Stack traces that led to blocking on synchronization primitives.',
  },
  'block:delay:nanoseconds:contentions:count': {
    name: 'Block Contention Time Total',
    help: 'Time delayed stack traces caused by blocking on synchronization primitives.',
  },
  'fgprof:samples:count:wallclock:nanoseconds:delta': {
    name: 'Fgprof Samples Total',
    help: 'CPU profile samples observed regardless of their current On/Off CPU scheduling status',
  },
  'fgprof:time:nanoseconds:wallclock:nanoseconds:delta': {
    name: 'Fgprof Samples Time Total',
    help: 'CPU profile measured regardless of their current On/Off CPU scheduling status in nanoseconds',
  },
  'goroutine:goroutine:count:goroutine:count': {
    name: 'Goroutine Created Total',
    help: 'Stack traces that created all current goroutines.',
  },
  'memory:alloc_objects:count:space:bytes': {
    name: 'Memory Allocated Objects Total',
    help: 'A sampling of all past memory allocations by objects.',
  },
  'memory:alloc_space:bytes:space:bytes': {
    name: 'Memory Allocated Bytes Total',
    help: 'A sampling of all past memory allocations in bytes.',
  },
  'memory:alloc_objects:count:space:bytes:delta': {
    name: 'Memory Allocated Objects Delta',
    help: 'A sampling of all memory allocations during the observation by objects.',
  },
  'memory:alloc_space:bytes:space:bytes:delta': {
    name: 'Memory Allocated Bytes Delta',
    help: 'A sampling of all memory allocations during the observation in bytes.',
  },
  'memory:inuse_objects:count:space:bytes': {
    name: 'Memory In-Use Objects',
    help: 'A sampling of memory allocations of live objects by objects.',
  },
  'memory:inuse_space:bytes:space:bytes': {
    name: 'Memory In-Use Bytes',
    help: 'A sampling of memory allocations of live objects by bytes.',
  },
  'mutex:contentions:count:contentions:count': {
    name: 'Mutex Contentions Total',
    help: 'Stack traces of holders of contended mutexes.',
  },
  'mutex:delay:nanoseconds:contentions:count': {
    name: 'Mutex Contention Time Total',
    help: 'Time delayed stack traces caused by contended mutexes.',
  },
  'process_cpu:cpu:nanoseconds:cpu:nanoseconds:delta': {
    name: 'Process CPU Nanoseconds',
    help: 'CPU profile measured by the process itself in nanoseconds.',
  },
  'process_cpu:samples:count:cpu:nanoseconds:delta': {
    name: 'Process CPU Samples',
    help: 'CPU profile samples observed by the process itself.',
  },
  'parca_agent_cpu:samples:count:cpu:nanoseconds:delta': {
    name: 'CPU Samples',
    help: 'CPU profile samples observed by Parca Agent.',
  },
  'otel_profiling_agent_on_cpu:samples:count:cpu:nanoseconds:delta': {
    name: 'On-CPU Samples',
    help: 'On CPU profile samples observed by the Otel Profiling Agent.',
  },
  'parca_agent:samples:count:cpu:nanoseconds:delta': {
    name: 'On-CPU',
    help: 'On CPU profile samples as observed by the Parca Agent.',
  },
  'parca_agent:wallclock:nanoseconds:samples:count:delta': {
    name: 'Off-CPU',
    help: 'Time spent off the CPU as observed by the Parca Agent.',
  },
  'parca_agent:cuda:nanoseconds:cuda:nanoseconds:delta': {
    name: 'On-GPU',
    help: 'Time spent on the GPU.',
  },
};

export function flexibleWellKnownProfileMatching(name: string): WellKnownProfile | undefined {
  const prefixExcludedName = name.split(':').slice(1).join(':');
  const deltaExcludedName = prefixExcludedName.replace(/:delta$/, '');
  const requiredKey = Object.keys(wellKnownProfiles).find(key => {
    if (key.includes(deltaExcludedName)) {
      return true;
    }
    return false;
  });
  return requiredKey != null ? wellKnownProfiles[requiredKey] : undefined;
}

export function profileSelectElement(
  name: string,
  flexibleKnownProfilesDetection: boolean
): SelectElement {
  const wellKnown: WellKnownProfile | undefined = !flexibleKnownProfilesDetection
    ? wellKnownProfiles[name]
    : flexibleWellKnownProfileMatching(name);
  if (wellKnown === undefined) return {active: <>{name}</>, expanded: <>{name}</>};

  const title = wellKnown.name.replace(/ /g, '\u00a0');
  return {
    active: <>{title}</>,
    expanded: (
      <>
        <span>{title}</span>
        <br />
        <span className="text-xs">{wellKnown.help}</span>
      </>
    ),
  };
}

export const constructProfileName = (type: ProfileType): string => {
  return `${type.name}:${type.sampleType}:${type.sampleUnit}:${type.periodType}:${type.periodUnit}${
    type.delta ? ':delta' : ''
  }`;
};

export const normalizeProfileTypesData = (types: ProfileType[]): string[] => {
  return types.map(constructProfileName).sort((a: string, b: string): number => {
    return a.localeCompare(b);
  });
};

interface Props {
  profileTypesData?: ProfileTypesResponse;
  loading?: boolean;
  error: RpcError | undefined;
  selectedKey: string | undefined;
  flexibleKnownProfilesDetection?: boolean;
  onSelection: (value: string | undefined) => void;
  disabled?: boolean;
}

const ProfileTypeSelector = ({
  profileTypesData,
  loading = false,
  error,
  selectedKey,
  onSelection,
  flexibleKnownProfilesDetection = false,
  disabled,
}: Props): JSX.Element => {
  const profileNames = useMemo(() => {
    return (error === undefined || error == null) &&
      profileTypesData !== undefined &&
      profileTypesData != null
      ? normalizeProfileTypesData(profileTypesData.types)
      : [];
  }, [profileTypesData, error]);

  const profileLabels = profileNames.map(name => ({
    key: name,
    element: profileSelectElement(name, flexibleKnownProfilesDetection),
  }));

  return (
    <Select
      items={profileLabels}
      selectedKey={selectedKey}
      onSelection={onSelection}
      placeholder="Select profile type..."
      loading={loading}
      className="bg-white h-profile-type-dropdown"
      disabled={disabled}
    />
  );
};

export default ProfileTypeSelector;
