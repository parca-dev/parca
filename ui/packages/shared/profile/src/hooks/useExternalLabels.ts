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

import {useQuery} from '@tanstack/react-query';

import {ExternalLabelSource} from '../contexts/UnifiedLabelsContext';

export const useExternalLabelValues = (
  labelName: string,
  externalLabelSource?: ExternalLabelSource
): {
  data: string[];
  loading: boolean;
  refetch: () => Promise<void>;
} => {
  const {data} = useQuery({
    queryKey: ['externalLabelValues', labelName],
    queryFn: async () => {
      const result = await externalLabelSource?.fetchLabelValues?.(labelName);
      return result ?? [];
    },
    enabled: externalLabelSource?.fetchLabelValues != null && labelName !== '',
  });

  return {
    data: data ?? [],
    loading: data == null,
    refetch: async () => {
      await externalLabelSource?.refetchLabelValues?.(labelName);
    },
  };
};
