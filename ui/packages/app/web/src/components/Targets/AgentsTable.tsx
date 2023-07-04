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

import {Agent} from '@parca/client';
import {TimeObject} from '@parca/utilities';

import LastScrapeCell from './LastScrapeCell';

const AgentsTableHeader = {
  id: 'Name',
  lastPush: 'Last Push',
  lastError: 'Last Error',
};

const getRowContentByHeader = ({
  header,
  agent,
  key,
}: {
  header: string;
  agent: Agent;
  key: string;
}) => {
  switch (header) {
    case AgentsTableHeader.id: {
      return (
        <td key={key} className="whitespace-nowrap px-6 py-4">
          {agent.id}
        </td>
      );
    }
    case AgentsTableHeader.lastError: {
      return (
        <td
          key={key}
          className="whitespace-nowrap px-6 py-4 text-sm text-gray-500 dark:text-gray-200"
        >
          {agent.lastError}
        </td>
      );
    }
    case AgentsTableHeader.lastPush: {
      const lastPush: TimeObject =
        agent.lastPush !== undefined
          ? {
              // Warning: string to number can overflow
              // https://github.com/timostamm/protobuf-ts/blob/master/MANUAL.md#bigint-support
              seconds: Number(agent.lastPush.seconds),
              nanos: agent.lastPush.nanos,
            }
          : {};
      const lastPushDuration: TimeObject =
        agent.lastPushDuration !== undefined
          ? {
              // Warning: string to number can overflow
              // https://github.com/timostamm/protobuf-ts/blob/master/MANUAL.md#bigint-support
              seconds: Number(agent.lastPushDuration.seconds),
              nanos: agent.lastPushDuration.nanos,
            }
          : {};
      return (
        <LastScrapeCell key={key} lastScrape={lastPush} lastScrapeDuration={lastPushDuration} />
      );
    }
    default: {
      return <td />;
    }
  }
};

const AgentsTable = ({agents}: {agents: Agent[]}) => {
  const headers = Object.keys(AgentsTableHeader) as (keyof typeof AgentsTableHeader)[];

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
              {AgentsTableHeader[header]}
            </th>
          ))}
        </tr>
      </thead>
      <tbody className="divide-y divide-gray-200 bg-white dark:divide-gray-700 dark:bg-gray-900">
        {agents.map((agent: Agent) => {
          return (
            <tr key={agent.id}>
              {headers.map(header => {
                const key = `table-cell-${header}-${agent.id}`;
                return getRowContentByHeader({header: AgentsTableHeader[header], agent, key});
              })}
            </tr>
          );
        })}
      </tbody>
    </table>
  );
};

export default AgentsTable;
