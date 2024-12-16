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

import {USER_PREFERENCES, useUserPreference} from '@parca/hooks';

interface Props {
  text: string;
  x: number;
  y: number;
  width: number;
}

function calculateTruncatedText(
  text: string,
  textElement: SVGTextElement,
  maxWidth: number
): string {
  // Create a temporary element for measurement
  const tempElement = textElement.cloneNode() as SVGTextElement;
  tempElement.textContent = text;
  textElement.parentElement?.appendChild(tempElement);

  // If the text fits, return it
  const fullWidth = tempElement.getComputedTextLength();
  if (fullWidth <= maxWidth) {
    textElement.parentElement?.removeChild(tempElement);
    return text;
  }

  // Binary search to find the maximum text that fits
  let start = 0;
  let end = text.length;
  let result = text;

  while (start < end) {
    const mid = Math.floor((start + end + 1) / 2);
    const truncated = text.slice(-mid);

    tempElement.textContent = truncated;
    const currentWidth = tempElement.getComputedTextLength();

    if (currentWidth <= maxWidth) {
      result = truncated;
      start = mid;
    } else {
      end = mid - 1;
    }
  }

  textElement.parentElement?.removeChild(tempElement);
  return result;
}

function TextWithEllipsis({text, x, y, width}: Props): JSX.Element {
  const textRef = useRef<SVGTextElement>(null);
  const [displayText, setDisplayText] = useState(text);
  const [showFunctionNameFromLeft] = useUserPreference<boolean>(
    USER_PREFERENCES.SHOW_FUNCTION_NAME_FROM_LEFT.key
  );

  useEffect(() => {
    const textElement = textRef.current;
    if (textElement === null) return;

    const newText = showFunctionNameFromLeft
      ? text
      : calculateTruncatedText(text, textElement, width);

    setDisplayText(newText);
  }, [text, width, showFunctionNameFromLeft]);

  return (
    <text ref={textRef} x={x} y={y} className="text-xs">
      {displayText}
    </text>
  );
}

export default TextWithEllipsis;
