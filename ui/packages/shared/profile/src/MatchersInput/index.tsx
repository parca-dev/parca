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

import React, {Fragment, useState, useEffect} from 'react';
import {Transition} from '@headlessui/react';
import {Query} from '@parca/parser';
import {LabelsResponse, QueryServiceClient} from '@parca/client';
import {usePopper} from 'react-popper';
import cx from 'classnames';

import {useParcaContext, useGrpcMetadata} from '@parca/components';
import SuggestionItem from './SuggestionItem';

interface MatchersInputProps {
  queryClient: QueryServiceClient;
  setMatchersString: (arg: string) => void;
  runQuery: () => void;
  currentQuery: Query;
}

export interface ILabelNamesResult {
  response?: LabelsResponse;
  error?: Error;
}

interface UseLabelNames {
  result: ILabelNamesResult;
  loading: boolean;
}

export const useLabelNames = (client: QueryServiceClient): UseLabelNames => {
  const [loading, setLoading] = useState(true);
  const [result, setResult] = useState<ILabelNamesResult>({});
  const metadata = useGrpcMetadata();

  useEffect(() => {
    const call = client.labels({match: []}, {meta: metadata});
    setLoading(true);

    call.response
      .then(response => setResult({response}))
      .catch(error => setResult({error}))
      .finally(() => setLoading(false));
  }, [client, metadata]);

  return {result, loading};
};

class Suggestion {
  type: string;
  typeahead: string;
  value: string;

  constructor(type: string, typeahead: string, value: string) {
    this.type = type;
    this.typeahead = typeahead;
    this.value = value;
  }
}

class Suggestions {
  literals: Suggestion[];
  labelNames: Suggestion[];
  labelValues: Suggestion[];

  constructor() {
    this.literals = [];
    this.labelNames = [];
    this.labelValues = [];
  }
}

const MatchersInput = ({
  queryClient,
  setMatchersString,
  runQuery,
  currentQuery,
}: MatchersInputProps): JSX.Element => {
  const [inputRef, setInputRef] = useState<HTMLInputElement | null>(null);
  const [focusedInput, setFocusedInput] = useState(false);
  const [showSuggest, setShowSuggest] = useState(true);
  const [highlightedSuggestionIndex, setHighlightedSuggestionIndex] = useState(-1);
  const [labelValuesLoading, setLabelValuesLoading] = useState(false);
  const [lastCompleted, setLastCompleted] = useState<Suggestion>(new Suggestion('', '', ''));
  const [popperElement, setPopperElement] = useState<HTMLDivElement | null>(null);
  const [labelValues, setLabelValues] = useState<string[] | null>(null);
  const [currentLabelName, setCurrentLabelName] = useState<string | null>(null);
  const {styles, attributes} = usePopper(inputRef, popperElement, {
    placement: 'bottom-start',
  });
  const metadata = useGrpcMetadata();
  const {loader: Spinner} = useParcaContext();

  const {loading: labelNamesLoading, result} = useLabelNames(queryClient);
  const {response: labelNamesResponse, error: labelNamesError} = result;

  const LoadingSpinner = (): JSX.Element => {
    return <div className="pt-2 pb-4">{Spinner}</div>;
  };

  useEffect(() => {
    if (currentLabelName !== null) {
      const call = queryClient.values({labelName: currentLabelName, match: []}, {meta: metadata});
      setLabelValuesLoading(true);

      call.response
        .then(response => {
          // replace single `\` in the `labelValues` string with doubles `\\` if available.
          const newValues = response.labelValues.map(value =>
            value.includes('\\') ? value.replace('\\', '\\\\') : value
          );

          setLabelValues(newValues);
        })
        .catch(() => setLabelValues(null))
        .finally(() => setLabelValuesLoading(false));
    }
  }, [currentLabelName, queryClient, metadata]);

  const labelNames =
    (labelNamesError === undefined || labelNamesError == null) &&
    labelNamesResponse !== undefined &&
    labelNamesResponse != null
      ? labelNamesResponse.labelNames.filter(e => e !== '__name__')
      : [];

  const value = currentQuery.matchersString();

  const suggestionSections = new Suggestions();
  Query.suggest(`${currentQuery.profileName()}{${value}`).forEach(function (s) {
    // Skip suggestions that we just completed. This really only works,
    // because we know the language is not repetitive. For a language that
    // has a repeating word, this would not work.
    if (lastCompleted !== null && lastCompleted.type === s.type) {
      return;
    }

    // Need to figure out if any literal suggestions make sense, but a
    // closing bracket doesn't in the guided query experience because all
    // we have the user do is type the matchers.
    if (s.type === 'literal' && s.value !== '}') {
      suggestionSections.literals.push({
        type: s.type,
        typeahead: s.typeahead,
        value: s.value,
      });
    }
    if (s.type === 'labelName') {
      const inputValue = s.typeahead.trim().toLowerCase();
      const inputLength = inputValue.length;
      const matches = labelNames.filter(function (label) {
        return label.toLowerCase().slice(0, inputLength) === inputValue;
      });

      matches.forEach(m =>
        suggestionSections.labelNames.push({
          type: s.type,
          typeahead: s.typeahead,
          value: m,
        })
      );
    }

    if (s.type === 'labelValue') {
      if (currentLabelName === null || s.labelName !== currentLabelName) {
        setCurrentLabelName(s.labelName);
        return;
      }

      if (labelValues !== null) {
        labelValues
          .filter(v => v.slice(0, s.typeahead.length) === s.typeahead)
          .forEach(v =>
            suggestionSections.labelValues.push({
              type: s.type,
              typeahead: s.typeahead,
              value: v,
            })
          );
      }
    }
  });

  const suggestionsLength =
    suggestionSections.literals.length +
    suggestionSections.labelNames.length +
    suggestionSections.labelValues.length;

  const resetHighlight = (): void => setHighlightedSuggestionIndex(-1);
  const resetLastCompleted = (): void => setLastCompleted(new Suggestion('', '', ''));

  const onChange = (e: React.ChangeEvent<HTMLInputElement>): void => {
    const newValue = e.target.value;
    setMatchersString(newValue);
    resetLastCompleted();
    resetHighlight();
  };

  const complete = (suggestion: Suggestion): string => {
    return value.slice(0, value.length - suggestion.typeahead.length) + suggestion.value;
  };

  const getSuggestion = (index: number): Suggestion => {
    if (index < suggestionSections.labelNames.length) {
      return suggestionSections.labelNames[index];
    }
    if (index < suggestionSections.labelNames.length + suggestionSections.literals.length) {
      return suggestionSections.literals[index - suggestionSections.labelNames.length];
    }
    return suggestionSections.labelValues[
      index - suggestionSections.labelNames.length - suggestionSections.literals.length
    ];
  };

  const highlightNext = (): void => {
    const nextIndex = highlightedSuggestionIndex + 1;
    if (nextIndex === suggestionsLength) {
      resetHighlight();
      return;
    }
    setHighlightedSuggestionIndex(nextIndex);
  };

  const highlightPrevious = (): void => {
    if (highlightedSuggestionIndex === -1) {
      // Didn't select anything, so starting at the bottom.
      setHighlightedSuggestionIndex(suggestionsLength - 1);
      return;
    }

    setHighlightedSuggestionIndex(highlightedSuggestionIndex - 1);
  };

  const applySuggestion = (suggestionIndex: number): void => {
    const suggestion = getSuggestion(suggestionIndex);
    const newValue = complete(suggestion);
    resetHighlight();
    setLastCompleted(suggestion);
    setMatchersString(newValue);
    if (inputRef !== null) {
      inputRef.value = newValue;
      inputRef.focus();
    }
  };

  const applyHighlightedSuggestion = (): void => {
    applySuggestion(highlightedSuggestionIndex);
  };

  const handleKeyPress = (event: React.KeyboardEvent<HTMLInputElement>): void => {
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
  };

  const handleKeyDown = (event: React.KeyboardEvent<HTMLInputElement>): void => {
    // Don't need to handle any key interactions if no suggestions there.
    if (suggestionsLength === 0) {
      return;
    }

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
  };

  const focus = (): void => {
    setFocusedInput(true);
  };

  const unfocus = (): void => {
    setFocusedInput(false);
    resetHighlight();
  };

  const profileSelected = currentQuery.profileName() === '';

  return (
    <div className="font-mono flex-1 w-full block">
      <input
        ref={setInputRef}
        type="text"
        className={cx(
          'bg-transparent focus:ring-indigo-800 flex-1 block w-full px-2 py-2 text-sm outline-none',
          profileSelected && 'cursor-not-allowed'
        )}
        placeholder={
          profileSelected
            ? 'Select a profile first to enter a filter...'
            : 'filter profiles... eg. node="test"'
        }
        onChange={onChange}
        value={value}
        onBlur={unfocus}
        onFocus={focus}
        onKeyPress={handleKeyPress}
        onKeyDown={handleKeyDown}
        disabled={profileSelected} // Disable input if no profile has been selected
        title={
          profileSelected
            ? 'Select a profile first to enter a filter...'
            : 'filter profiles... eg. node="test"'
        }
      />
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
              className="absolute z-10 max-h-[400px] mt-1 bg-gray-50 dark:bg-gray-900 shadow-lg rounded-md text-base ring-1 ring-black ring-opacity-5 overflow-auto focus:outline-none sm:text-sm"
            >
              {labelNamesLoading ? (
                <LoadingSpinner />
              ) : (
                <>
                  {suggestionSections.labelNames.map((l, i) => (
                    <SuggestionItem
                      isHighlighted={highlightedSuggestionIndex === i}
                      onHighlight={() => setHighlightedSuggestionIndex(i)}
                      onApplySuggestion={() => applySuggestion(i)}
                      onResetHighlight={() => resetHighlight()}
                      value={l.value}
                      key={l.value}
                    />
                  ))}
                </>
              )}

              {suggestionSections.literals.map((l, i) => (
                <SuggestionItem
                  isHighlighted={
                    highlightedSuggestionIndex === i + suggestionSections.labelNames.length
                  }
                  onHighlight={() =>
                    setHighlightedSuggestionIndex(i + suggestionSections.labelNames.length)
                  }
                  onApplySuggestion={() =>
                    applySuggestion(i + suggestionSections.labelNames.length)
                  }
                  onResetHighlight={() => resetHighlight()}
                  value={l.value}
                  key={l.value}
                />
              ))}

              {labelValuesLoading && lastCompleted.type === 'literal' ? (
                <LoadingSpinner />
              ) : (
                <>
                  {suggestionSections.labelValues.map((l, i) => (
                    <SuggestionItem
                      isHighlighted={
                        highlightedSuggestionIndex ===
                        i +
                          suggestionSections.labelNames.length +
                          suggestionSections.literals.length
                      }
                      onHighlight={() =>
                        setHighlightedSuggestionIndex(
                          i +
                            suggestionSections.labelNames.length +
                            suggestionSections.literals.length
                        )
                      }
                      onApplySuggestion={() =>
                        applySuggestion(
                          i +
                            suggestionSections.labelNames.length +
                            suggestionSections.literals.length
                        )
                      }
                      onResetHighlight={() => resetHighlight()}
                      value={l.value}
                      key={l.value}
                    />
                  ))}
                </>
              )}
            </div>
          </Transition>
        </div>
      )}
    </div>
  );
};

export default MatchersInput;
