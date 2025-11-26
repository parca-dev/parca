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

import React, {useMemo, useRef, useState} from 'react';

import cx from 'classnames';
import TextareaAutosize from 'react-textarea-autosize';

import {Query} from '@parca/parser';
import {TEST_IDS, testId} from '@parca/test-utils';

import {useUnifiedLabels} from '../contexts/UnifiedLabelsContext';
import {useQueryState} from '../hooks/useQueryState';
import SuggestionsList, {Suggestion, Suggestions} from './SuggestionsList';

const MatchersInput = (): JSX.Element => {
  const inputRef = useRef<HTMLTextAreaElement | null>(null);
  const [focusedInput, setFocusedInput] = useState(false);
  const [lastCompleted, setLastCompleted] = useState<Suggestion>(new Suggestion('', '', ''));

  const {
    labelNames,
    labelValues,
    labelNameMappingsForMatchersInput: labelNameMappings,
    isLabelNamesLoading,
    isLabelValuesLoading,
    currentLabelName,
    setCurrentLabelName,
    shouldHandlePrefixes,
    refetchLabelValues,
    refetchLabelNames,
    suffix,
  } = useUnifiedLabels();

  const {setDraftMatchers, commitDraft, draftParsedQuery} = useQueryState({suffix});

  const value = draftParsedQuery != null ? draftParsedQuery.matchersString() : '';

  const suggestionSections = useMemo(() => {
    const suggestionSections = new Suggestions();
    Query.suggest(`${draftParsedQuery?.profileName() as string}{${value}`).forEach(function (s) {
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

        matches.forEach(m => {
          const suggestion = {
            type: s.type,
            typeahead: s.typeahead,
            value: m,
          };

          if (shouldHandlePrefixes) {
            const mapping = labelNameMappings.find(l => l.displayName === m);
            if (mapping != null) {
              (suggestion as any).fullName = mapping.fullName;
            }
          }

          suggestionSections.labelNames.push(suggestion);
        });
      }

      if (s.type === 'labelValue') {
        let labelNameForQuery = s.labelName;

        if (shouldHandlePrefixes) {
          const mapping = labelNameMappings.find(l => l.displayName === s.labelName);
          if (mapping != null) {
            labelNameForQuery = mapping.fullName;
          }
        }

        if (currentLabelName === null || labelNameForQuery !== currentLabelName) {
          setCurrentLabelName(labelNameForQuery);
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
    return suggestionSections;
  }, [
    draftParsedQuery,
    lastCompleted,
    labelNames,
    labelValues,
    currentLabelName,
    value,
    shouldHandlePrefixes,
    labelNameMappings,
    setCurrentLabelName,
  ]);

  const resetLastCompleted = (): void => setLastCompleted(new Suggestion('', '', ''));

  const onChange = (e: React.ChangeEvent<HTMLTextAreaElement>): void => {
    const newValue = e.target.value;
    setDraftMatchers(newValue);
    resetLastCompleted();
  };

  const complete = (suggestion: Suggestion): string => {
    let newValue = value.slice(0, value.length - suggestion.typeahead.length) + suggestion.value;

    // Add a starting quote if we're completing a operator literal
    if (suggestion.type === 'literal' && suggestion.value !== ',') {
      newValue += '"';
    }

    // Add a closing quote if we're completing a label value
    if (suggestion.type === 'labelValue') {
      newValue += '"';
    }

    return newValue;
  };

  const applySuggestion = (suggestion: Suggestion): void => {
    const newValue = complete(suggestion);
    setLastCompleted(suggestion);
    setDraftMatchers(newValue);
    if (inputRef.current !== null) {
      inputRef.current.value = newValue;
      inputRef.current.focus();
    }
  };

  const focus = (): void => {
    setFocusedInput(true);
  };

  const unfocus = (): void => {
    setFocusedInput(false);
  };

  const profileSelected = draftParsedQuery?.profileName() === '';

  return (
    <div
      className="w-full min-w-[300px] flex-1 font-mono relative"
      {...testId(TEST_IDS.MATCHERS_INPUT_CONTAINER)}
    >
      <TextareaAutosize
        ref={inputRef}
        className={cx(
          'block h-[38px] w-full flex-1 rounded-md border bg-white px-2 py-2 text-sm shadow-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500 dark:border-gray-600 dark:bg-gray-900',
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
        {...testId(TEST_IDS.MATCHERS_TEXTAREA)}
        onFocus={focus}
        disabled={profileSelected} // Disable input if no profile has been selected
        title={
          profileSelected
            ? 'Select a profile first to enter a filter...'
            : 'filter profiles... eg. node="test"'
        }
        id="matchers-input"
      />
      <SuggestionsList
        isLabelNamesLoading={isLabelNamesLoading}
        suggestions={suggestionSections}
        applySuggestion={applySuggestion}
        inputRef={inputRef.current}
        runQuery={commitDraft}
        focusedInput={focusedInput}
        isLabelValuesLoading={
          isLabelValuesLoading && lastCompleted.type === 'literal' && lastCompleted.value !== ','
        }
        shouldTrimPrefix={shouldHandlePrefixes}
        refetchLabelValues={refetchLabelValues}
        refetchLabelNames={refetchLabelNames}
      />
    </div>
  );
};

export default MatchersInput;
