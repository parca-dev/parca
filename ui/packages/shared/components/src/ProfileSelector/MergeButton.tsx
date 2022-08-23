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

import React, {useState} from 'react';
import {usePopper} from 'react-popper';
import {Button} from '../';

const MergeButton = ({disabled, onClick}: {disabled: boolean; onClick: () => void}) => {
  const [mergeHover, setMergeHover] = useState<boolean>(false);
  const [mergePopperReferenceElement, setMergePopperReferenceElement] =
    useState<HTMLDivElement | null>(null);
  const [mergePopperElement, setMergePopperElement] = useState<HTMLDivElement | null>(null);
  const {styles, attributes} = usePopper(mergePopperReferenceElement, mergePopperElement, {
    placement: 'bottom',
  });

  const mergeExplanation =
    'Merging allows combining all profile samples of a query into a single report.';

  if (disabled) return <></>;

  return (
    <div ref={setMergePopperReferenceElement}>
      <Button
        color="neutral"
        disabled={disabled}
        onClick={onClick}
        onMouseEnter={() => setMergeHover(true)}
        onMouseLeave={() => setMergeHover(false)}
      >
        Merge
      </Button>
      {mergeHover && (
        <div
          ref={setMergePopperElement}
          style={styles.popper}
          {...attributes.popper}
          className="z-50"
        >
          <div className="flex">
            <div className="relative mx-2">
              <svg className="text-black h-1 w-full left-0" x="0px" y="0px" viewBox="0 0 255 127.5">
                <polygon className="fill-current" points="0,127.5 127.5,0 255,127.5" />
              </svg>
              <div className="bg-black text-white text-xs rounded py-2 px-3 right-0 w-40 z-50">
                {mergeExplanation}
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default MergeButton;
