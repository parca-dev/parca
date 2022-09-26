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
import {LabelsResponse, QueryServiceClient, ValuesResponse} from '@parca/client';
import {usePopper} from 'react-popper';
import cx from 'classnames';

import {useParcaTheme, useGrpcMetadata} from '@parca/components';

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
export interface ILabelValuesResult {
  response?: ValuesResponse;
  error?: Error;
}

interface UseLabelNames {
  result: ILabelNamesResult;
  loading: boolean;
}

interface Matchers {
  key: string;
  matcherType: string;
  value: string;
}

enum Labels {
  labelName = 'labelName',
  labelValue = 'labelValue',
  literal = 'literal',
}

// eslint-disable-next-line no-useless-escape
const labelNameValueRe = /(^([a-z])\w+)(=~|=|!=|!~)(\")[a-zA-Z0-9_.-:]+(\")$/g; // labelNameValueRe matches the following: labelName=~"labelValue"
const labelNameValueWithoutQuotesRe = /(^([a-z])\w+)(=~|=|!=|!~)[a-zA-Z0-9_.-:]+$/g; // labelNameValueWithoutQuotesRe matches the following: labelName=~labelValue
const labelNameLiteralRe = /(^([a-z])\w+)(=~|=|!=|!~)/; // labelNameLiteralRe matches the following: labelName=~, labelName!=~, labelName=, labelName!=
const literalRe = /(=~|=|!=|!~)/; // literalRe matches the following: =~, =, !=, !~

const addQuoteMarks = (labelValue: string): string => {
  // eslint-disable-next-line no-useless-escape
  return `\"${labelValue}\"`;
};

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
  const [divInputRef, setDivInputRef] = useState<HTMLDivElement | null>(null);
  const [popperElement, setPopperElement] = useState<HTMLDivElement | null>(null);
  const [localMatchers, setLocalMatchers] = useState<Matchers[] | null>(null);
  const [focusedInput, setFocusedInput] = useState(false);
  const [showSuggest, setShowSuggest] = useState(true);
  const [labelValuesLoading, setLabelValuesLoading] = useState(false);
  const [highlightedSuggestionIndex, setHighlightedSuggestionIndex] = useState(-1);
  const [lastCompleted, setLastCompleted] = useState<Suggestion>(new Suggestion('', '', ''));
  const [suggestionSections] = useState<Suggestions>(new Suggestions());
  const [inputRef, setInputRef] = useState<string>('');
  const [labelValuesResponse, setLabelValuesResponse] = useState<string[] | null>(null);
  const [currentLabelsCollection, setCurrentLabelsCollection] = useState<string[] | null>(null); // This is an array that contains query expressions that have been matched i.e. they have been completed and have the blue badge around them in the UI.
  const {styles, attributes} = usePopper(divInputRef, popperElement, {
    placement: 'bottom-start',
  });
  const metadata = useGrpcMetadata();
  const {loader: Spinner} = useParcaTheme();

  const {loading: labelNamesLoading, result} = useLabelNames(queryClient);
  const {response: labelNamesResponse, error: labelNamesError} = result;

  const LoadingSpinner = (): JSX.Element => {
    return <div className="pt-2 pb-4">{Spinner}</div>;
  };

  const getLabelNameValues = (labelName: string): void => {
    const call = queryClient.values({labelName, match: []}, {meta: metadata});
    setLabelValuesLoading(true);

    call.response
      .then(response => {
        setLabelValuesResponse(response.labelValues);
      })
      .catch(() => setLabelValuesResponse(null))
      .finally(() => setLabelValuesLoading(false));
  };

  const labelNames =
    (labelNamesError === undefined || labelNamesError == null) &&
    labelNamesResponse !== undefined &&
    labelNamesResponse != null
      ? labelNamesResponse.labelNames.filter(e => e !== '__name__')
      : [];

  const labelValues =
    labelValuesResponse !== undefined && labelValuesResponse != null ? labelValuesResponse : [];

  const value = currentQuery.matchersString();

  Query.suggest(`{${value}`).forEach(function (s) {
    // Skip suggestions that we just completed. This really only works,
    // because we know the language is not repetitive. For a language that
    // has a repeating word, this would not work.
    if (lastCompleted !== null && lastCompleted.type === s.type) {
      return;
    }

    // Need to figure out if any literal suggestions make sense, but a
    // closing bracket doesn't in the guided query experience because all
    // we have the user do is type the matchers.
    if (s.type === Labels.literal && s.value !== '}') {
      if (suggestionSections.literals.find(e => e.value === s.value) != null) {
        return;
      }
      suggestionSections.literals.push({
        type: s.type,
        typeahead: '',
        value: s.value,
      });
      suggestionSections.labelNames = [];
      suggestionSections.labelValues = [];
    }

    if (s.type === Labels.labelName) {
      const inputValue = s.typeahead.trim().toLowerCase();
      const inputLength = inputValue.length;

      const matches = labelNames.filter(function (label) {
        return label.toLowerCase().slice(0, inputLength) === inputValue;
      });

      matches.forEach(m => {
        if (suggestionSections.labelNames.find(e => e.value === m) != null) {
          return;
        }

        suggestionSections.labelNames.push({
          type: s.type,
          typeahead: s.typeahead,
          value: m,
        });
        suggestionSections.literals = [];
        suggestionSections.labelValues = [];
      });
    }

    if (s.type === Labels.labelValue) {
      const inputValue = s.typeahead.trim().toLowerCase();
      const inputLength = inputValue.length;

      const matches = labelValues.filter(function (label) {
        return label.toLowerCase().slice(0, inputLength) === inputValue;
      });

      matches.forEach(m => {
        if (suggestionSections.labelValues.find(e => e.value === m) != null) {
          return;
        }

        suggestionSections.labelValues.push({
          type: s.type,
          typeahead: s.typeahead,
          value: m,
        });
        suggestionSections.labelNames = [];
        suggestionSections.literals = [];
      });
    }
  });

  const suggestionsLength =
    suggestionSections.literals.length +
    suggestionSections.labelNames.length +
    suggestionSections.labelValues.length;

  const getLabelsFromMatchers = (matchers: Matchers[]): string[] => {
    return matchers
      .filter(matcher => matcher.key !== '__name__')
      .map(matcher => `${matcher.key}${matcher.matcherType}${addQuoteMarks(matcher.value)}`);
  };

  useEffect(() => {
    const matchers = currentQuery.matchers.filter(matcher => matcher.key !== '__name__');

    if (matchers.length > 0) {
      setCurrentLabelsCollection(getLabelsFromMatchers(matchers));
    } else {
      if (localMatchers !== null) setCurrentLabelsCollection(getLabelsFromMatchers(localMatchers));
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [currentQuery.matchers]);

  const resetHighlight = (): void => setHighlightedSuggestionIndex(-1);
  const resetLastCompleted = (): void => setLastCompleted(new Suggestion('', '', ''));

  const onChange = (e: React.ChangeEvent<HTMLInputElement>): void => {
    const newValue = e.target.value;

    // filter out the labelname list and move to the top the labelname that is most similar to what the user is typing.
    if (suggestionSections.labelNames.length > 0) {
      suggestionSections.labelNames = suggestionSections.labelNames.filter(suggestion =>
        suggestion.value.toLowerCase().includes(newValue.toLowerCase())
      );
    }

    // this checks if the user has typed a label name and a literal (=/!=,=~,!~) i.e labelName=~, labelName!=~, labelName=, labelName!=
    // and is about to type the label value, then it it will filter out the labelvalue list and move to the top
    // the labelvalue that is most similar to what the user is typing.
    if (suggestionSections.labelValues.length > 0 && labelNameLiteralRe.test(newValue)) {
      const labelValueSearch = newValue.split(literalRe)[2];

      suggestionSections.labelValues = suggestionSections.labelValues.filter(suggestion =>
        suggestion.value.toLowerCase().includes(labelValueSearch.toLowerCase())
      );
    }

    setInputRef(newValue);
    resetLastCompleted();
    resetHighlight();
  };

  const complete = (suggestion: Suggestion): string => {
    return value.slice(0, value.length - suggestion.typeahead.length) + suggestion.value;
  };

  const getSuggestion = (index: number): Suggestion => {
    if (suggestionSections.labelValues.length > 0) {
      if (index < suggestionSections.labelValues.length) {
        return suggestionSections.labelValues[index];
      }
      return suggestionSections.literals[index - suggestionSections.labelValues.length];
    }

    if (index < suggestionSections.labelNames.length) {
      return suggestionSections.labelNames[index];
    }
    return suggestionSections.literals[index - suggestionSections.labelNames.length];
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

    if (suggestion.type === Labels.labelValue) {
      suggestion.value = addQuoteMarks(suggestion.value);
    }

    const newValue = complete(suggestion);
    resetHighlight();

    if (suggestion.type === Labels.labelName) {
      getLabelNameValues(suggestion.value);
    }

    setLastCompleted(suggestion);
    setMatchersString(newValue);

    if (suggestion.type === Labels.labelValue) {
      const values = newValue.split(',');

      if (currentLabelsCollection == null || currentLabelsCollection?.length === 0) {
        setCurrentLabelsCollection(values);
      } else {
        setCurrentLabelsCollection((oldValues: string[] | null) => [
          ...(oldValues ?? []),
          values[values.length - 1],
        ]);
      }

      setInputRef('');
      focus();
      return;
    }

    if (lastCompleted.type === Labels.labelValue && suggestion.type === Labels.literal) {
      setInputRef('');
      focus();
      return;
    }

    if (currentLabelsCollection !== null) {
      setInputRef(newValue.substring(newValue.lastIndexOf(',') + 1));
      focus();
      return;
    }

    setInputRef(newValue);
    focus();
  };

  const applyHighlightedSuggestion = (): void => {
    applySuggestion(highlightedSuggestionIndex);
  };

  // This function adds quotes to the query expression if the user has typed it in manually, i.e. did not use the arrow up / down keys + Enter
  // to choose the label name and value. Therefore, labelName=value becomes labelName="value".
  const addQuotesToInputRefLabelValue = (inputRef: string): string => {
    const labelValue = inputRef.split(literalRe)[2].replaceAll(',', '');
    const labelValueWithQuotes = addQuoteMarks(labelValue);
    return inputRef.replace(labelValue, labelValueWithQuotes);
  };

  const handleKeyUp = (event: React.KeyboardEvent<HTMLInputElement>): void => {
    const values = inputRef.replaceAll(',', '');

    if (labelNameValueRe.test(inputRef)) {
      if (currentLabelsCollection === null) {
        setMatchersString(inputRef);
      } else {
        setMatchersString(currentLabelsCollection?.join(',') + ',' + values);
      }
      setInputRef('');
    }

    if (event.key === ',') {
      if (inputRef.length === 0) event.preventDefault();

      // If the current typed query expression matches the labelNameValueWithoutQuotesRe regex (i.e. the labelvalue is not quoted), then add quotes to the labelvalue.
      // if not, just use the current inputRef value.
      const inputValues = labelNameValueWithoutQuotesRe.test(inputRef)
        ? inputRef.replaceAll(',', '')
        : addQuotesToInputRefLabelValue(inputRef).replaceAll(',', '');

      // if the currentLabelsCollection array is null, we don't need to concat the current inputRef value with the currentLabelsCollection array, so we just push to it.
      if (currentLabelsCollection === null) {
        setCurrentLabelsCollection([inputValues]);
      } else {
        setCurrentLabelsCollection((oldValues: string[] | null) => {
          // Don't add the current inputRef value to the currentLabelsCollection array if it doesn't match the regex because that will cause an API error.
          if (!labelNameValueRe.test(inputRef)) return oldValues;
          return [...(oldValues ?? []), inputValues];
        });
      }

      // update the currentQuery expression with the currentLabelsCollection array if it's not null, otherwise use the current inputRef value.
      setMatchersString(
        currentLabelsCollection !== null
          ? `${currentLabelsCollection?.join(',')},${inputValues}`
          : `${inputValues},`
      );
      setInputRef('');
    }

    // We suggest the appropriate label names and label values when a user is typing, depending on what the user has typed.
    // For example, if the user types "labelName=", we suggest the label values next.
    // This bit of code is used for the opposite of the above bit of code, when a user is deleting characters by pressing del/backspace
    // We update the currentQuery expression with what's in the inputRef value so that the suggestions are updated accordingly.
    if (event.key === 'Backspace' && inputRef.length > 0) {
      // if the currentLabelsCollection array is not empty i.e has already previously completed expressions, then we first need to turn the array into a string
      // so it can be concatenated with the current inputRef value. that becomes something like "labelName="value",newLabelName="val
      if (currentLabelsCollection != null && currentLabelsCollection.length > 0) {
        setMatchersString(`${currentLabelsCollection?.join(',')},${inputRef}}`);
      } else {
        // if not, we jsut update the currentQuery expression with the current inputRef value.
        setMatchersString(inputRef);
      }

      if (currentLabelsCollection === null && inputRef.length === 0) {
        setMatchersString('');
      }
    }
  };

  const handleKeyPress = (event: React.KeyboardEvent<HTMLInputElement>): void => {
    // If there is a highlighted suggestion and enter is hit, we complete
    // with the highlighted suggestion.
    if (highlightedSuggestionIndex >= 0 && event.key === 'Enter') {
      applyHighlightedSuggestion();
      if (lastCompleted.type === Labels.labelValue) setLabelValuesResponse(null);

      const matchers = currentQuery.matchers.filter(matcher => matcher.key !== '__name__');
      setLocalMatchers(prevState => {
        if (inputRef.length > 0) return prevState;
        if (matchers.length === 0) return prevState;
        return matchers;
      });
    }

    // If a user has manually typed in a label name that actually exists in the list of label name (and did not use the
    // highlight + arrow keys up/down + Enter/Mouse click method to complete it), and has also typed a literal value, i.e. labelName=,
    // then we can apply a suggestion using the typed label name. This will be as if the user had highlighted the label name and hit enter.
    if (event.key === '!' || event.key === '~' || event.key === '=') {
      const labelName = inputRef.split(literalRe)[0];

      if (suggestionSections.labelNames.length > 0) {
        // Find the label name in the suggestion list and get the index
        const suggestion = suggestionSections.labelNames.find(
          suggestion => suggestion.value === labelName
        );
        // If the typed label name exists, we can apply it using the applySuggestion function
        if (suggestion != null) {
          applySuggestion(suggestionSections.labelNames.indexOf(suggestion));
        }
      }
    }

    // Same as above, If a user has typed in a label name and literal (and did not use the suggestion box to complete it),
    // we can manually show the next set of suggestions, which are the label values, by applying a literal suggestion.
    if (labelNameLiteralRe.test(inputRef)) {
      const literal = inputRef.split(literalRe)[1];

      if (suggestionSections.literals.length > 0) {
        // Find the literal in the suggestion list and get the index
        const suggestion = suggestionSections.literals.find(
          suggestion => suggestion.value === literal
        );
        // If the typed literal exists, we can apply it using the applySuggestion function
        if (suggestion != null) {
          applySuggestion(suggestionSections.literals.indexOf(suggestion));
        }
      }
    }

    // If no suggestions is highlighted and we hit enter, we run the query,
    // and hide suggestions until another actions enables them again.
    if (highlightedSuggestionIndex === -1 && event.key === 'Enter') {
      if (lastCompleted.type === 'labelValue') setLabelValuesResponse(null);
      setShowSuggest(false);
      runQuery();
      return;
    }

    setShowSuggest(true);
  };

  const handleKeyDown = (event: React.KeyboardEvent<HTMLInputElement>): void => {
    if (event.key === 'Backspace' && inputRef === '') {
      if (currentLabelsCollection === null) return;

      removeLabel(currentLabelsCollection.length - 1);
      removeLocalMatcher();
    }

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

  const removeLabel = (label: number): void => {
    if (currentLabelsCollection === null) return;

    const newLabels = [...currentLabelsCollection];
    newLabels.splice(label, 1);
    setCurrentLabelsCollection(newLabels);

    const newLabelsAsAString = newLabels.join(',');
    setMatchersString(newLabelsAsAString);
  };

  const removeLocalMatcher = (): void => {
    if (localMatchers === null) return;

    const newMatchers = [...localMatchers];
    newMatchers.splice(localMatchers.length - 1, 1);
    setLocalMatchers(newMatchers);
  };

  const profileSelected = currentQuery.profType.profileName === '';

  return (
    <>
      <div
        ref={setDivInputRef}
        className="w-full flex items-center text-sm border-gray-300 dark:border-gray-600 border-b"
      >
        <ul className="flex space-x-2">
          {currentLabelsCollection?.map((value, i) => (
            <li
              key={i}
              className="bg-indigo-600 w-fit py-1 px-2 text-gray-100 dark-gray-900 rounded-md"
            >
              {value}
            </li>
          ))}
        </ul>

        <input
          type="text"
          className={cx(
            'bg-transparent focus:ring-indigo-800 flex-1 block w-full px-2 py-2 text-sm outline-none',
            profileSelected && 'cursor-not-allowed'
          )}
          placeholder={
            profileSelected ? 'Select a profile first to query profiles...' : 'query profiles...'
          }
          onChange={onChange}
          value={inputRef}
          onBlur={unfocus}
          onFocus={focus}
          onKeyPress={handleKeyPress}
          onKeyDown={handleKeyDown}
          onKeyUp={handleKeyUp}
          disabled={profileSelected} // Disable input if no profile has been selected
          title={
            profileSelected ? 'Select a profile first to query profiles...' : 'query profiles...'
          }
        />
      </div>

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
            style={{width: divInputRef?.offsetWidth}}
            className="absolute z-10 max-h-[400px] mt-1 bg-gray-50 dark:bg-gray-900 shadow-lg rounded-md text-base ring-1 ring-black ring-opacity-5 overflow-auto focus:outline-none sm:text-sm"
          >
            {labelNamesLoading ? (
              <LoadingSpinner />
            ) : (
              <>
                {suggestionSections.labelNames.map((l, i) => (
                  <div
                    key={i}
                    className={cx(
                      highlightedSuggestionIndex === i && 'text-white bg-indigo-600',
                      'cursor-default select-none relative py-2 pl-3 pr-9'
                    )}
                    onMouseOver={() => setHighlightedSuggestionIndex(i)}
                    onClick={() => applySuggestion(i)}
                    onMouseOut={() => resetHighlight()}
                  >
                    {l.value}
                  </div>
                ))}
              </>
            )}

            {suggestionSections.literals.map((l, i) => (
              <div
                key={i}
                className={cx(
                  highlightedSuggestionIndex === i + suggestionSections.labelNames.length &&
                    'text-white bg-indigo-600',
                  'cursor-default select-none relative py-2 pl-3 pr-9'
                )}
                onMouseOver={() =>
                  setHighlightedSuggestionIndex(i + suggestionSections.labelNames.length)
                }
                onClick={() => applySuggestion(i + suggestionSections.labelNames.length)}
                onMouseOut={() => resetHighlight()}
              >
                {l.value}
              </div>
            ))}

            {labelValuesLoading && lastCompleted.type === 'literal' ? (
              <LoadingSpinner />
            ) : (
              <>
                {suggestionSections.labelValues.map((l, i) => (
                  <div
                    key={i}
                    className={cx(
                      highlightedSuggestionIndex === i && 'text-white bg-indigo-600',
                      'cursor-default select-none relative py-2 pl-3 pr-9'
                    )}
                    onMouseOver={() => setHighlightedSuggestionIndex(i)}
                    onClick={() => applySuggestion(i)}
                    onMouseOut={() => resetHighlight()}
                  >
                    {l.value}
                  </div>
                ))}
              </>
            )}
          </div>
        </Transition>
      </div>
    </>
  );
};

export default MatchersInput;
