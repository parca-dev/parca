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

import cx from 'classnames';

interface Props {
  isHighlighted: boolean;
  onHighlight: () => void;
  onApplySuggestion: () => void;
  onResetHighlight: () => void;
  value: string;
}

const SuggestionItem = ({
  isHighlighted,
  onHighlight,
  onApplySuggestion,
  onResetHighlight,
  value,
}: Props): JSX.Element => {
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (isHighlighted && ref.current != null) {
      ref.current.scrollIntoView({block: 'nearest'});
    }
  }, [isHighlighted]);

  return (
    <div
      className={cx('relative cursor-default select-none py-2 pl-3 pr-9', {
        'bg-indigo-600 text-white': isHighlighted,
      })}
      onMouseOver={() => onHighlight()}
      onClick={() => onApplySuggestion()}
      onMouseOut={() => onResetHighlight()}
      ref={ref}
    >
      {value}
    </div>
  );
};

export default SuggestionItem;
