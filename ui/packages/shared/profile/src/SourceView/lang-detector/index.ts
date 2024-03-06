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

import * as hljsLangs from 'react-syntax-highlighter/dist/cjs/languages/hljs';

import _extLangMap from './ext-to-lang.json';

const extLangMap = _extLangMap as Record<string, string[]>;

export const langaugeFromFile = (file: string): string => {
  const extension = file.split('.').pop() ?? '';
  if (extLangMap[extension] == null) {
    return 'text';
  }
  const langs: string[] = extLangMap[extension];
  for (const lang of langs) {
    // eslint-disable-next-line import/namespace
    if (hljsLangs[lang] != null) {
      return lang;
    }
  }
  return 'text';
};
