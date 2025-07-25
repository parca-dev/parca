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

import React, {ReactNode, useCallback, useEffect, useRef, useState} from 'react';

import cx from 'classnames';

interface Props {
  children: ReactNode;
  disabled?: boolean;
  placeholder?: string;
}

const MatchersInputMask = ({
  children,
  disabled = false,
  placeholder = 'Select label names and values to filter with...',
}: Props): JSX.Element => {
  const [isFocused, setIsFocused] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);

  const handleFocusIn = useCallback(() => {
    if (!disabled) {
      setIsFocused(true);
    }
  }, [disabled]);

  const handleFocusOut = useCallback((e: React.FocusEvent) => {
    // Only blur if focus is moving completely outside the mask container
    if (e.relatedTarget === null || !e.currentTarget.contains(e.relatedTarget as Node)) {
      setIsFocused(false);
    }
  }, []);

  // Handle clicks outside the component to remove focus
  useEffect(() => {
    const handleDocumentClick = (event: MouseEvent): void => {
      if (containerRef.current !== null && !containerRef.current.contains(event.target as Node)) {
        setIsFocused(false);
      }
    };

    if (isFocused) {
      document.addEventListener('mousedown', handleDocumentClick);
    }

    return () => {
      document.removeEventListener('mousedown', handleDocumentClick);
    };
  }, [isFocused]);

  return (
    <div
      ref={containerRef}
      className={cx(
        'w-full min-w-[300px] flex-1 relative rounded-md border bg-white shadow-sm transition-colors min-h-[38px] flex items-center',
        'dark:border-gray-600 dark:bg-gray-900',
        {
          'border-indigo-500 ring-1 ring-indigo-500': isFocused && !disabled,
          'border-gray-300 dark:border-gray-600': !isFocused && !disabled,
          'border-gray-200 bg-gray-50 dark:border-gray-700 dark:bg-gray-800 cursor-not-allowed':
            disabled,
        }
      )}
      onFocusCapture={handleFocusIn}
      onBlurCapture={handleFocusOut}
      role="textbox"
      aria-disabled={disabled}
      aria-placeholder={placeholder}
    >
      {children}
    </div>
  );
};

export default MatchersInputMask;
