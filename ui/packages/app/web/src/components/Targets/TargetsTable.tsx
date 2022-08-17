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
import LabelsCell from './LabelsCell';
import LastScrapeCell from './LastScrapeCell';
import {getHealthStatus} from './utils';
import {Pill} from '@parca/components';

enum TargetsTableHeader {
  url = 'URL',
  health = 'Health Status',
  labels = 'Labels',
  lastScrape = 'Last Scrape',
  lastError = 'Last Error',
}

const getRowContentByHeader = ({
  header,
  target,
  key,
}: {
  header: string;
  target: any;
  key: string;
}) => {
  switch (header) {
    case TargetsTableHeader.url: {
      const {url} = target;
      return (
        <td key={key} className="px-6 py-4 whitespace-nowrap">
          <a
            className="text-sm text-gray-900 text-bold dark:text-gray-200"
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
      const {
        labels: {labels},
        discoveredLabels: {labels: discoveredLabels},
      } = target;
      return <LabelsCell labels={labels} discoveredLabels={discoveredLabels} key={key} />;
    }
    case TargetsTableHeader.lastError: {
      const {lastError} = target;
      return (
        <td
          key={key}
          className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-200"
        >
          {lastError}
        </td>
      );
    }
    case TargetsTableHeader.lastScrape: {
      const {lastScrape, lastScrapeDuration} = target;
      return (
        <LastScrapeCell key={key} lastScrape={lastScrape} lastScrapeDuration={lastScrapeDuration} />
      );
    }
    case TargetsTableHeader.health: {
      const {health} = target;
      const {label, colorVariant} = getHealthStatus(health);
      return (
        <td key={key} className="px-6 py-4 whitespace-nowrap">
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
          {headers.map((header: string) => (
            <th
              key={header}
              scope="col"
              className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-200 uppercase tracking-wider"
            >
              {TargetsTableHeader[header]}
            </th>
          ))}
        </tr>
      </thead>
      <tbody className="bg-white divide-y divide-gray-200 dark:bg-gray-900 dark:divide-gray-700">
        {targets.map((target: Target) => {
          return (
            <tr key={target.url}>
              {headers.map((header: string) => {
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
