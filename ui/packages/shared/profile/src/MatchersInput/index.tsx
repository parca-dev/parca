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

import {LabelsRequest, LabelsResponse, QueryServiceClient, ValuesRequest} from '@parca/client';
import {useGrpcMetadata} from '@parca/components';
import {Query} from '@parca/parser';
import {millisToProtoTimestamp, sanitizeLabelValue} from '@parca/utilities';

import useGrpcQuery from '../useGrpcQuery';
import SuggestionsList, {Suggestion, Suggestions} from './SuggestionsList';

interface MatchersInputProps {
  queryClient: QueryServiceClient;
  setMatchersString: (arg: string) => void;
  runQuery: () => void;
  currentQuery: Query;
  profileType: string;
}

export interface ILabelNamesResult {
  response?: LabelsResponse;
  error?: Error;
}

interface UseLabelNames {
  result: ILabelNamesResult;
  loading: boolean;
}

export const useLabelNames = (
  client: QueryServiceClient,
  profileType: string,
  start?: number,
  end?: number,
  match?: string[]
): UseLabelNames => {
  const metadata = useGrpcMetadata();

  const {data, isLoading, error} = useGrpcQuery<LabelsResponse>({
    key: ['labelNames', profileType, match?.join(',')],
    queryFn: async () => {
      const request: LabelsRequest = {match: match !== undefined ? match : []};
      if (start !== undefined && end !== undefined) {
        request.start = millisToProtoTimestamp(start);
        request.end = millisToProtoTimestamp(end);
      }
      if (profileType !== undefined) {
        request.profileType = profileType;
      }
      const {response} = await client.labels(request, {meta: metadata});
      return response;
    },
    options: {
      enabled: profileType !== undefined && profileType !== '',
      staleTime: 1000 * 60 * 5, // 5 minutes
      keepPreviousData: false,
    },
  });

  return {result: {response: data, error: error as Error}, loading: isLoading};
};

interface UseLabelValues {
  result: {
    response: string[];
    error?: Error;
  };
  loading: boolean;
}

export const useLabelValues = (
  client: QueryServiceClient,
  labelName: string,
  profileType: string
): UseLabelValues => {
  const metadata = useGrpcMetadata();

  const {data, isLoading, error} = useGrpcQuery<string[]>({
    key: ['labelValues', labelName, profileType],
    queryFn: async () => {
      const request: ValuesRequest = {labelName, match: [], profileType};
      const {response} = await client.values(request, {meta: metadata});
      return sanitizeLabelValue(response.labelValues);
    },
    options: {
      enabled:
        profileType !== undefined &&
        profileType !== '' &&
        labelName !== undefined &&
        labelName !== '',
      staleTime: 1000 * 60 * 5, // 5 minutes
      keepPreviousData: false,
    },
  });

  return {result: {response: data ?? [], error: error as Error}, loading: isLoading};
};

const MatchersInput = ({
  queryClient,
  setMatchersString,
  runQuery,
  currentQuery,
  profileType,
}: MatchersInputProps): JSX.Element => {
  const inputRef = useRef<HTMLTextAreaElement | null>(null);
  const [focusedInput, setFocusedInput] = useState(false);
  const [lastCompleted, setLastCompleted] = useState<Suggestion>(new Suggestion('', '', ''));
  const [currentLabelName, setCurrentLabelName] = useState<string | null>(null);

  const {loading: labelNamesLoading, result} = useLabelNames(queryClient, profileType);
  const {response: labelNamesResponse, error: labelNamesError} = result;

  const {
    loading: labelValuesLoading,
    result: {response: labelValues},
  } = useLabelValues(queryClient, currentLabelName ?? '', profileType);

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
    <div className="w-full min-w-[300px] flex-1 font-mono relative">
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
