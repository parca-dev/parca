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

import {createParser, parseAsArrayOf, parseAsBoolean, parseAsInteger, parseAsString} from 'nuqs';

const opts = {history: 'replace' as const};

// === Base parsers with common options ===
export const stringParam = parseAsString.withOptions(opts);
export const boolParam = parseAsBoolean.withOptions(opts);
export const intParam = parseAsInteger.withOptions(opts);
export const commaArrayParam = parseAsArrayOf(parseAsString, ',').withOptions(opts);

// === Param-specific parsers with defaults ===
export const colorByParser = stringParam.withDefault('binary');
export const invertCallStackParser = boolParam.withDefault(false);
export const dashboardItemsParser = commaArrayParam.withDefault(['flamegraph']);
export const groupByParser = commaArrayParam;
export const flamechartDimensionParser = commaArrayParam;
export const tableColumnsParser = commaArrayParam;
export const hiddenBinariesParser = commaArrayParam.withDefault([]);

// === JSON parser with BigInt support ===
export const jsonParser = <T>() =>
  createParser<T>({
    parse: (value: string) => JSON.parse(value) as T,
    serialize: (value: T) =>
      JSON.stringify(value, (_, v) => (typeof v === 'bigint' ? v.toString() : v)),
  }).withOptions(opts);
