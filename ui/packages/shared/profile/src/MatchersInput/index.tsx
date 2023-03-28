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

import React, {useEffect, useMemo, useRef, useState} from 'react';

import cx from 'classnames';
import TextareaAutosize from 'react-textarea-autosize';

import {LabelsResponse, QueryServiceClient} from '@parca/client';
import {useGrpcMetadata} from '@parca/components';
import {Query} from '@parca/parser';
import {sanitizeLabelValue} from '@parca/utilities';

import SuggestionsList, {Suggestion, Suggestions} from './SuggestionsList';

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

const MatchersInput = ({
  queryClient,
  setMatchersString,
  runQuery,
  currentQuery,
}: MatchersInputProps): JSX.Element => {
  const inputRef = useRef<HTMLTextAreaElement | null>(null);
  const [focusedInput, setFocusedInput] = useState(false);
  const [labelValuesLoading, setLabelValuesLoading] = useState(false);
  const [lastCompleted, setLastCompleted] = useState<Suggestion>(new Suggestion('', '', ''));
  const [labelValues, setLabelValues] = useState<string[] | null>(null);
  const [currentLabelName, setCurrentLabelName] = useState<string | null>(null);
  const metadata = useGrpcMetadata();

  const {loading: labelNamesLoading, result} = useLabelNames(queryClient);
  const {response: labelNamesResponse, error: labelNamesError} = result;

  useEffect(() => {
    if (currentLabelName !== null) {
      const call = queryClient.values({labelName: currentLabelName, match: []}, {meta: metadata});
      setLabelValuesLoading(true);

      call.response
        .then(response => {
          // replace single `\` in the `labelValues` string with doubles `\\` if available.
          const newValues = sanitizeLabelValue(response.labelValues);

          return setLabelValues(newValues);
        })
        .catch(() => setLabelValues(null))
        .finally(() => setLabelValuesLoading(false));
    }
  }, [currentLabelName, queryClient, metadata]);

  const labelNames = useMemo(() => {
    return (labelNamesError === undefined || labelNamesError == null) &&
      labelNamesResponse !== undefined &&
      labelNamesResponse != null
      ? labelNamesResponse.labelNames.filter(e => e !== '__name__')
      : [];
  }, [labelNamesError, labelNamesResponse]);

  const value = currentQuery.matchersString();

  const suggestionSections = useMemo(() => {
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
    return suggestionSections;
  }, [currentQuery, lastCompleted, labelNames, labelValues, currentLabelName, value]);

  const resetLastCompleted = (): void => setLastCompleted(new Suggestion('', '', ''));

  const onChange = (e: React.ChangeEvent<HTMLTextAreaElement>): void => {
    const newValue = e.target.value;
    setMatchersString(newValue);
    resetLastCompleted();
  };

  const complete = (suggestion: Suggestion): string => {
    return value.slice(0, value.length - suggestion.typeahead.length) + suggestion.value;
  };

  const applySuggestion = (suggestion: Suggestion): void => {
    const newValue = complete(suggestion);
    setLastCompleted(suggestion);
    setMatchersString(newValue);
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

  const profileSelected = currentQuery.profileName() === '';

  return (
    <div className="w-full min-w-[300px] flex-1 font-mono">
      <TextareaAutosize
        ref={inputRef}
        className={cx(
          'block w-full flex-1 rounded bg-gray-50 px-2 py-2 text-sm outline-none focus:ring-indigo-800 dark:bg-gray-900',
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
        disabled={profileSelected} // Disable input if no profile has been selected
        title={
          profileSelected
            ? 'Select a profile first to enter a filter...'
            : 'filter profiles... eg. node="test"'
        }
      />
      <SuggestionsList
        isLabelNamesLoading={labelNamesLoading}
        suggestions={suggestionSections}
        applySuggestion={applySuggestion}
        inputRef={inputRef.current}
        runQuery={runQuery}
        focusedInput={focusedInput}
        isLabelValuesLoading={labelValuesLoading && lastCompleted.type === 'literal'}
      />
    </div>
  );
};

export default MatchersInput;
