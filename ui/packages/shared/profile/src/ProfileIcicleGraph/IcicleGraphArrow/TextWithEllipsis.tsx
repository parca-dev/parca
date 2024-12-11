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

import {useEffect, useRef, useState} from 'react';

import cx from 'classnames';

import {USER_PREFERENCES, useUserPreference} from '@parca/hooks';

interface TextWithEllipsisProps {
  text: string;
  x: number;
  y: number;
  width: number;
}

function TextWithEllipsis({text, x, y, width}: TextWithEllipsisProps): React.ReactNode {
  const textRef = useRef<SVGTextElement>(null);
  const [displayText, setDisplayText] = useState(text);

  const [showFunctionNameFromLeft] = useUserPreference<boolean>(
    USER_PREFERENCES.SHOW_FUNCTION_NAME_FROM_LEFT.key
  );

  useEffect(() => {
    if (showFunctionNameFromLeft) {
      setDisplayText(text);
      return;
    }

    const textElement = textRef.current;
    if (textElement === null) return;

    const textWidth = textElement.getComputedTextLength();
    if (textWidth <= width) {
      setDisplayText(text);
      return;
    }

    // Binary search to find the maximum text that fits
    let start = 0;
    let end = text.length;
    let result = text;

    while (start < end) {
      const mid = Math.floor((start + end + 1) / 2);
      const truncated = !showFunctionNameFromLeft
        ? `...${text.slice(-mid)}`
        : `${text.slice(0, mid)}...`;

      textElement.textContent = truncated;
      const currentWidth = textElement.getComputedTextLength();

      if (currentWidth <= width) {
        result = truncated;
        start = mid;
      } else {
        end = mid - 1;
      }
    }

    setDisplayText(result);
  }, [text, width, showFunctionNameFromLeft]);

  if (showFunctionNameFromLeft) {
    return (
      <text ref={textRef} x={x} y={y} className="text-xs">
        {displayText}
      </text>
    );
  }

  return (
    <text ref={textRef} x={x} y={y} className="text-xs">
      {displayText}
    </text>
  );
}

export default TextWithEllipsis;
