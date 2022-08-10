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

import intervalToDuration from 'date-fns/intervalToDuration';

export const formatForTimespan = (from: number, to: number): string => {
  const duration = intervalToDuration({start: from, end: to});
  if (duration <= {minutes: 61}) {
    return 'H:mm';
  }
  if (duration <= {hours: 13}) {
    return 'H';
  }
  if (duration <= {hours: 25}) {
    return 'H:mm d/M';
  }
  return 'd/M';
};
