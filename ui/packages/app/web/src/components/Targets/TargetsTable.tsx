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

import React from 'react';

import {Target} from '@parca/client';
import {Pill} from '@parca/components';
import {TimeObject} from '@parca/utilities';

import LabelsCell from './LabelsCell';
import LastScrapeCell from './LastScrapeCell';
import {getHealthStatus} from './utils';

const TargetsTableHeader = {
  url: 'URL',
  health: 'Health Status',
  labels: 'Labels',
  lastScrape: 'Last Scrape',
  lastError: 'Last Error',
};

const getRowContentByHeader = ({
  header,
  target,
  key,
}: {
  header: string;
  target: Target;
  key: string;
}) => {
  switch (header) {
    case TargetsTableHeader.url: {
      const {url} = target;
      return (
        <td key={key} className="whitespace-nowrap px-6 py-4">
          <a
            className="text-bold text-sm text-gray-900 dark:text-gray-200"
            href={url}
            target="_blank"
            rel="noreferrer"
          >
            {url}
          </a>
        </td>
      );
    }
    case TargetsTableHeader.labels: {
      const labels = target.labels?.labels ?? [];
      const discoveredLabels = target.discoveredLabels?.labels ?? [];
      return <LabelsCell labels={labels} discoveredLabels={discoveredLabels} key={key} />;
    }
    case TargetsTableHeader.lastError: {
      const {lastError} = target;
      return (
        <td
          key={key}
          className="whitespace-nowrap px-6 py-4 text-sm text-gray-500 dark:text-gray-200"
        >
          {lastError}
        </td>
      );
    }
    case TargetsTableHeader.lastScrape: {
      const lastScrape: TimeObject =
        target.lastScrape !== undefined
          ? {
              // Warning: string to number can overflow
              // https://github.com/timostamm/protobuf-ts/blob/master/MANUAL.md#bigint-support
              seconds: Number(target.lastScrape.seconds),
              nanos: target.lastScrape.nanos,
            }
          : {};
      const lastScrapeDuration: TimeObject =
        target.lastScrapeDuration !== undefined
          ? {
              // Warning: string to number can overflow
              // https://github.com/timostamm/protobuf-ts/blob/master/MANUAL.md#bigint-support
              seconds: Number(target.lastScrapeDuration.seconds),
              nanos: target.lastScrapeDuration.nanos,
            }
          : {};
      return (
        <LastScrapeCell key={key} lastScrape={lastScrape} lastScrapeDuration={lastScrapeDuration} />
      );
    }
    case TargetsTableHeader.health: {
      const {health} = target;
      const {label, colorVariant} = getHealthStatus(health);

      return (
        <td key={key} className="whitespace-nowrap px-6 py-4">
          <Pill variant={colorVariant}>{label}</Pill>
        </td>
      );
    }
    default: {
      return <td />;
    }
  }
};

const TargetsTable = ({targets}: {targets: Target[]}) => {
  const headers = Object.keys(TargetsTableHeader) as Array<keyof typeof TargetsTableHeader>;

  return (
    <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
      <thead className="bg-gray-50 dark:bg-gray-800">
        <tr>
          {headers.map(header => (
            <th
              key={header}
              scope="col"
              className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-gray-200"
            >
              {TargetsTableHeader[header]}
            </th>
          ))}
        </tr>
      </thead>
      <tbody className="divide-y divide-gray-200 bg-white dark:divide-gray-700 dark:bg-gray-900">
        {targets.map((target: Target) => {
          return (
            <tr key={target.url}>
              {headers.map(header => {
                const key = `table-cell-${header}-${target.url}`;
                return getRowContentByHeader({header: TargetsTableHeader[header], target, key});
              })}
            </tr>
          );
        })}
      </tbody>
    </table>
  );
};

export default TargetsTable;
