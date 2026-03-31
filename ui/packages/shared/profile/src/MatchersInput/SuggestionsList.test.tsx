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

import React, {useRef} from 'react';

import {act, fireEvent, render, screen} from '@testing-library/react';
import {beforeAll, describe, expect, it, vi} from 'vitest';

import SuggestionsList, {Suggestion, Suggestions} from './SuggestionsList';

vi.mock('@parca/components', () => ({
  RefreshButton: ({title}: {title: string}) => <button type="button">{title}</button>,
  useParcaContext: () => ({
    loader: <div>loading</div>,
  }),
}));

beforeAll(() => {
  Element.prototype.scrollIntoView = vi.fn();
});

const TestHarness = ({inputKey = 'initial'}: {inputKey?: string}): JSX.Element => {
  const inputRef = useRef<HTMLTextAreaElement | null>(null);
  const suggestions = new Suggestions();
  suggestions.labelNames.push(new Suggestion('labelName', 'na', 'namespace'));

  return (
    <div>
      <textarea key={inputKey} ref={inputRef} />
      <SuggestionsList
        suggestions={suggestions}
        applySuggestion={vi.fn()}
        inputRef={inputRef}
        runQuery={vi.fn()}
        focusedInput
        isLabelNamesLoading={false}
        isLabelValuesLoading={false}
        shouldTrimPrefix={false}
        refetchLabelValues={vi.fn(async () => {})}
        refetchLabelNames={vi.fn(async () => {})}
      />
    </div>
  );
};

describe('SuggestionsList', () => {
  it('rebinds keyboard listeners when the textarea ref points to a remounted node', () => {
    const {rerender} = render(<TestHarness inputKey="first" />);

    rerender(<TestHarness inputKey="second" />);

    const textarea = screen.getByRole('textbox');
    act(() => {
      fireEvent.keyDown(textarea, {key: 'ArrowDown'});
    });

    expect(screen.getByText('namespace').className).toContain('bg-indigo-600');
  });
});
