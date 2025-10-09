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

import {Fragment, useCallback, useEffect, useState} from 'react';

import {Transition} from '@headlessui/react';
import {Icon} from '@iconify/react';
import cx from 'classnames';
import {usePopper} from 'react-popper';

import {useParcaContext} from '@parca/components';

import SuggestionItem from './SuggestionItem';

export class Suggestion {
  type: string;
  typeahead: string;
  value: string;

  constructor(type: string, typeahead: string, value: string) {
    this.type = type;
    this.typeahead = typeahead;
    this.value = value;
  }
}

export class Suggestions {
  literals: Suggestion[];
  labelNames: Suggestion[];
  labelValues: Suggestion[];

  constructor() {
    this.literals = [];
    this.labelNames = [];
    this.labelValues = [];
  }
}

interface Props {
  suggestions: Suggestions;
  applySuggestion: (suggestion: Suggestion) => void;
  inputRef: HTMLTextAreaElement | null;
  runQuery: () => void;
  focusedInput: boolean;
  isLabelNamesLoading: boolean;
  isLabelValuesLoading: boolean;
  shouldTrimPrefix: boolean;
  refetchLabelValues: () => void;
}

const LoadingSpinner = (): JSX.Element => {
  const {loader: Spinner} = useParcaContext();

  return <div className="pt-2 pb-4">{Spinner}</div>;
};

const transformLabelsForSuggestions = (labelNames: string, shouldTrimPrefix = false): string => {
  const trimmedLabel = shouldTrimPrefix ? labelNames.split('.').pop() ?? labelNames : labelNames;
  return trimmedLabel;
};

const SuggestionsList = ({
  suggestions,
  applySuggestion,
  inputRef,
  runQuery,
  focusedInput,
  isLabelNamesLoading,
  isLabelValuesLoading,
  shouldTrimPrefix = false,
  refetchLabelValues,
}: Props): JSX.Element => {
  const [popperElement, setPopperElement] = useState<HTMLDivElement | null>(null);
  const {styles, attributes} = usePopper(inputRef, popperElement, {
    placement: 'bottom-start',
  });
  const [highlightedSuggestionIndex, setHighlightedSuggestionIndex] = useState<number>(-1);
  const [showSuggest, setShowSuggest] = useState(true);
  const [isRefetching, setIsRefetching] = useState(false);

  const handleRefetch = useCallback(async () => {
    if (isRefetching) return;

    setIsRefetching(true);
    try {
      await refetchLabelValues();
    } finally {
      setIsRefetching(false);
    }
  }, [refetchLabelValues, isRefetching]);

  const suggestionsLength =
    suggestions.literals.length + suggestions.labelNames.length + suggestions.labelValues.length;

  const getSuggestion = useCallback(
    (index: number): Suggestion => {
      if (index < suggestions.labelNames.length) {
        return suggestions.labelNames[index];
      }
      if (index < suggestions.labelNames.length + suggestions.literals.length) {
        return suggestions.literals[index - suggestions.labelNames.length];
      }
      return suggestions.labelValues[
        index - suggestions.labelNames.length - suggestions.literals.length
      ];
    },
    [suggestions]
  );

  const resetHighlight = useCallback(
    (): void => setHighlightedSuggestionIndex(-1),
    [setHighlightedSuggestionIndex]
  );

  const applyHighlightedSuggestion = useCallback((): void => {
    applySuggestion(getSuggestion(highlightedSuggestionIndex));
    resetHighlight();
  }, [resetHighlight, applySuggestion, highlightedSuggestionIndex, getSuggestion]);

  const applySuggestionWithIndex = useCallback(
    (index: number): void => {
      applySuggestion(getSuggestion(index));
      resetHighlight();
    },
    [resetHighlight, applySuggestion, getSuggestion]
  );

  const highlightNext = useCallback((): void => {
    const nextIndex = highlightedSuggestionIndex + 1;
    if (nextIndex === suggestionsLength) {
      resetHighlight();
      return;
    }
    setHighlightedSuggestionIndex(nextIndex);
  }, [highlightedSuggestionIndex, suggestionsLength, resetHighlight]);

  const highlightPrevious = useCallback((): void => {
    if (highlightedSuggestionIndex === -1) {
      // Didn't select anything, so starting at the bottom.
      setHighlightedSuggestionIndex(suggestionsLength - 1);
      return;
    }

    setHighlightedSuggestionIndex(highlightedSuggestionIndex - 1);
  }, [highlightedSuggestionIndex, suggestionsLength]);

  const handleKeyPress = useCallback(
    (event: React.KeyboardEvent<HTMLTextAreaElement>): void => {
      if (event.key === 'Enter') {
        // Disable new line in the text area
        event.preventDefault();
      }
      // If there is a highlighted suggestion and enter is hit, we complete
      // with the highlighted suggestion.
      if (highlightedSuggestionIndex >= 0 && event.key === 'Enter') {
        applyHighlightedSuggestion();
      }

      // If no suggestions is highlighted and we hit enter, we run the query,
      // and hide suggestions until another actions enables them again.
      if (highlightedSuggestionIndex === -1 && event.key === 'Enter') {
        setShowSuggest(false);
        runQuery();
        return;
      }

      setShowSuggest(true);
    },
    [highlightedSuggestionIndex, applyHighlightedSuggestion, runQuery]
  );

  const handleKeyDown = useCallback(
    (event: KeyboardEvent): void => {
      // Don't need to handle any key interactions if no suggestions there.
      if (suggestionsLength === 0 || !['Tab', 'ArrowUp', 'ArrowDown'].includes(event.key)) {
        return;
      }

      event.preventDefault();

      // Handle tabbing through suggestions.
      if (event.key === 'Tab' && suggestionsLength > 0) {
        event.preventDefault();
        if (event.shiftKey) {
          // Shift + tab goes up.
          highlightPrevious();
          return;
        }
        // Just tab goes down.
        highlightNext();
      }

      // Up arrow highlights previous suggestions.
      if (event.key === 'ArrowUp') {
        highlightPrevious();
      }

      // Down arrow highlights next suggestions.
      if (event.key === 'ArrowDown') {
        highlightNext();
      }
    },
    [suggestionsLength, highlightNext, highlightPrevious]
  );

  useEffect(() => {
    if (inputRef == null) {
      return;
    }

    inputRef.addEventListener('keydown', handleKeyDown);
    inputRef.addEventListener('keypress', handleKeyPress as any);

    return () => {
      inputRef.removeEventListener('keydown', handleKeyDown);
      inputRef.removeEventListener('keypress', handleKeyPress as any);
    };
  }, [inputRef, highlightedSuggestionIndex, suggestions, handleKeyPress, handleKeyDown]);

  return (
    <>
      {suggestionsLength > 0 && (
        <div
          ref={setPopperElement}
          style={{...styles.popper, marginLeft: 0}}
          {...attributes.popper}
          className="z-50"
        >
          <Transition
            show={focusedInput && showSuggest}
            as={Fragment}
            leave="transition ease-in duration-100"
            leaveFrom="opacity-100"
            leaveTo="opacity-0"
          >
            <div
              style={{width: inputRef?.offsetWidth}}
              className="absolute z-10 mt-1 max-h-[400px] overflow-auto rounded-md bg-gray-50 text-base shadow-lg ring-1 ring-black ring-opacity-5 focus:outline-none dark:bg-gray-900 sm:text-sm"
            >
              <div className="relative pb-12">
                {isLabelNamesLoading ? (
                  <LoadingSpinner />
                ) : (
                  <>
                    {suggestions.labelNames.map((l, i) => (
                      <SuggestionItem
                        isHighlighted={highlightedSuggestionIndex === i}
                        onHighlight={() => setHighlightedSuggestionIndex(i)}
                        onApplySuggestion={() => applySuggestionWithIndex(i)}
                        onResetHighlight={() => resetHighlight()}
                        value={transformLabelsForSuggestions(l.value, shouldTrimPrefix)}
                        key={transformLabelsForSuggestions(l.value, shouldTrimPrefix)}
                      />
                    ))}
                  </>
                )}

                {suggestions.literals.map((l, i) => (
                  <SuggestionItem
                    isHighlighted={highlightedSuggestionIndex === i + suggestions.labelNames.length}
                    onHighlight={() =>
                      setHighlightedSuggestionIndex(i + suggestions.labelNames.length)
                    }
                    onApplySuggestion={() =>
                      applySuggestionWithIndex(i + suggestions.labelNames.length)
                    }
                    onResetHighlight={() => resetHighlight()}
                    value={l.value}
                    key={l.value}
                  />
                ))}

                {isLabelValuesLoading ? (
                  <LoadingSpinner />
                ) : suggestions.labelNames.length === 0 && suggestions.literals.length === 0 ? (
                  <>
                    {suggestions.labelValues.length === 0 ? (
                      <div
                        className="px-4 py-3 text-sm text-gray-500 dark:text-gray-400 text-center"
                        data-testid="suggestions-no-results"
                      >
                        No label values found
                      </div>
                    ) : (
                      suggestions.labelValues.map((l, i) => (
                        <SuggestionItem
                          isHighlighted={
                            highlightedSuggestionIndex ===
                            i + suggestions.labelNames.length + suggestions.literals.length
                          }
                          onHighlight={() =>
                            setHighlightedSuggestionIndex(
                              i + suggestions.labelNames.length + suggestions.literals.length
                            )
                          }
                          onApplySuggestion={() =>
                            applySuggestionWithIndex(
                              i + suggestions.labelNames.length + suggestions.literals.length
                            )
                          }
                          onResetHighlight={() => resetHighlight()}
                          value={l.value}
                          key={l.value}
                        />
                      ))
                    )}
                    <div className="absolute w-full flex items-center justify-center bottom-0 px-3 py-2 bg-gray-50 dark:bg-gray-800">
                      <button
                        onClick={e => {
                          e.preventDefault();
                          e.stopPropagation();
                          void handleRefetch();
                        }}
                        disabled={isRefetching}
                        className={cx(
                          'p-1 flex items-center gap-1 rounded-full transition-all duration-200 w-auto justify-center',
                          isRefetching
                            ? 'cursor-wait opacity-50'
                            : 'hover:bg-gray-200 dark:hover:bg-gray-700 cursor-pointer'
                        )}
                        title="Refresh label values"
                        type="button"
                        data-testid="suggestions-refresh-button"
                      >
                        <Icon
                          icon="system-uicons:reset"
                          className={cx(
                            'w-3 h-3 text-gray-500 dark:text-gray-400',
                            isRefetching && 'animate-spin'
                          )}
                        />
                        <span className="text-xs text-gray-500 dark:text-gray-400">
                          Refresh results
                        </span>
                      </button>
                    </div>
                  </>
                ) : (
                  suggestions.labelValues.map((l, i) => (
                    <SuggestionItem
                      isHighlighted={
                        highlightedSuggestionIndex ===
                        i + suggestions.labelNames.length + suggestions.literals.length
                      }
                      onHighlight={() =>
                        setHighlightedSuggestionIndex(
                          i + suggestions.labelNames.length + suggestions.literals.length
                        )
                      }
                      onApplySuggestion={() =>
                        applySuggestionWithIndex(
                          i + suggestions.labelNames.length + suggestions.literals.length
                        )
                      }
                      onResetHighlight={() => resetHighlight()}
                      value={l.value}
                      key={l.value}
                    />
                  ))
                )}
              </div>
            </div>
          </Transition>
        </div>
      )}
    </>
  );
};

export default SuggestionsList;
