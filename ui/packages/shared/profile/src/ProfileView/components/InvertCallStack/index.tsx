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

import {Icon} from '@iconify/react';

import {Button, useURLState} from '@parca/components';

const InvertCallStack = (): JSX.Element => {
  const [invertStack = '', setInvertStack] = useURLState('invert_call_stack');
  const isInvert = invertStack === 'true';

  return (
    <div className="flex flex-col">
      <label className="text-sm">&nbsp;</label>
      <Button
        variant="neutral"
        className="flex items-center gap-2 whitespace-nowrap"
        onClick={() => setInvertStack(isInvert ? '' : 'true')}
        id="h-invert-call-stack"
      >
        <Icon icon={isInvert ? 'ph:sort-ascending' : 'ph:sort-descending'} className="h-4 w-4" />
        {isInvert ? 'Original' : 'Invert'} Call Stack
      </Button>
    </div>
  );
};

export default InvertCallStack;
