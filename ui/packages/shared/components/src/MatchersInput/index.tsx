import React, {Fragment, useState, useEffect} from 'react';
import {Transition} from '@headlessui/react';
import {Query} from '@parca/parser';
import {
  LabelsResponse,
  LabelsRequest,
  QueryServiceClient,
  ServiceError,
  ValuesRequest,
  ValuesResponse,
} from '@parca/client';
import {usePopper} from 'react-popper';
import cx from 'classnames';

interface MatchersInputProps {
  queryClient: QueryServiceClient;
  setMatchersString: (arg: string) => void;
  runQuery: () => void;
  currentQuery: Query;
}

export interface ILabelNamesResult {
  response: LabelsResponse.AsObject | null;
  error: ServiceError | null;
}
export interface ILabelValuesResult {
  response: ValuesResponse.AsObject | null;
  error: ServiceError | null;
}

const pasteSplit = data => {
  const separators = [',', ';', '\\(', '\\)', '\\*', '/', ':', '\\?', '\n', '\r'];
  return data.split(new RegExp(separators.join('|'))).map(d => d.trim());
};

const addQuoteMarks = (labelValue: string) => {
  // eslint-disable-next-line no-useless-escape
  return `\"${labelValue}\"`;
};

export const useLabelNames = (client: QueryServiceClient): ILabelNamesResult => {
  const [result, setResult] = useState<ILabelNamesResult>({
    response: null,
    error: null,
  });

  useEffect(() => {
    client.labels(
      new LabelsRequest(),
      (error: ServiceError | null, responseMessage: LabelsResponse | null) => {
        const res = responseMessage == null ? null : responseMessage.toObject();

        setResult({
          response: res,
          error: error,
        });
      }
    );
  }, [client]);

  return result;
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
  const [divInputRef, setDivInputRef] = useState<HTMLDivElement | null>(null);
  const [currentInputValue, setCurrentInputValue] = useState<any>(null);
  const [focusedInput, setFocusedInput] = useState(false);
  const [showSuggest, setShowSuggest] = useState(true);
  const [highlightedSuggestionIndex, setHighlightedSuggestionIndex] = useState(-1);
  const [lastCompleted, setLastCompleted] = useState<Suggestion>(new Suggestion('', '', ''));
  const [popperElement, setPopperElement] = useState<HTMLDivElement | null>(null);
  const [labelValuesResponse, setLabelValuesResponse] = useState<string[] | null>(null);
  const {styles, attributes} = usePopper(inputRef, popperElement, {
    placement: 'bottom-start',
  });

  const {response: labelNamesResponse, error: labelNamesError} = useLabelNames(queryClient);

  const getLabelNameValues = (labelName: string) => {
    const req = new ValuesRequest();
    req.setLabelName(labelName);

    queryClient.values(
      req,
      (error: ServiceError | null, responseMessage: ValuesResponse | null) => {
        setLabelValuesResponse(
          responseMessage == null ? null : responseMessage.toObject().labelValuesList
        );
      }
    );
  };

  const labelNames =
    (labelNamesError === undefined || labelNamesError == null) &&
    labelNamesResponse !== undefined &&
    labelNamesResponse != null
      ? labelNamesResponse.labelNamesList.filter(e => e !== '__name__')
      : [];

  const labelValues =
    labelValuesResponse !== undefined && labelValuesResponse != null ? labelValuesResponse : [];

  const value = currentQuery.matchersString();
  const suggestionSections = new Suggestions();

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
    if (s.type === 'literal' && s.value !== '}') {
      suggestionSections.literals.push({
        type: s.type,
        typeahead: '',
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
      const inputValue = s.typeahead.trim().toLowerCase();
      const inputLength = inputValue.length;

      const matches = labelValues.filter(function (label) {
        return label.toLowerCase().slice(0, inputLength) === inputValue;
      });

      matches.forEach(m =>
        suggestionSections.labelValues.push({
          type: s.type,
          typeahead: s.typeahead,
          value: m,
        })
      );
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
    console.log('🚀 ~ file: MatchersInput.tsx ~ line 198 ~ onChange ~ newValue', newValue);
    setMatchersString(newValue);
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
    let suggestion = getSuggestion(suggestionIndex);

    if (suggestion.type === 'labelValue') {
      suggestion.value = addQuoteMarks(suggestion.value);
      inputRef.value = '';
    }

    const newValue = complete(suggestion);
    resetHighlight();

    if (suggestion.type === 'labelName') {
      getLabelNameValues(suggestion.value);
    }

    setLastCompleted(suggestion);
    setMatchersString(newValue);

    if (suggestion.type === 'labelValue') {
      setCurrentInputValue(newValue.split(','));
    }

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
      if (lastCompleted.type === 'labelValue') setLabelValuesResponse(null);
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

  return (
    <>
      <div
        ref={setDivInputRef}
        className="w-full flex items-center text-sm border-gray-300 dark:border-gray-600 border-b"
      >
        <ul>{currentInputValue && currentInputValue.map((value, i) => <li>{value}</li>)}</ul>

        <input
          ref={setInputRef}
          type="text"
          className="bg-transparent focus:ring-indigo-800 flex-1 block w-full px-2 py-2 text-sm outline-none"
          placeholder="filter profiles..."
          onChange={onChange}
          value={value}
          onBlur={unfocus}
          onFocus={focus}
          onKeyPress={handleKeyPress}
          onKeyDown={handleKeyDown}
        />
      </div>

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
              className="absolute z-10 mt-1 bg-gray-50 dark:bg-gray-900 shadow-lg rounded-md text-base ring-1 ring-black ring-opacity-5 overflow-auto focus:outline-none sm:text-sm"
            >
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
            </div>
          </Transition>
        </div>
      )}
    </>
  );
};

export default MatchersInput;
