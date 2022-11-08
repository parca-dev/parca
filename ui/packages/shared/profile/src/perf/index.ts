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

import afterFrame from 'afterframe';
import {ProfilerOnRenderCallback} from 'react';

interface PerfLogObject {
  id: string;
  phase: string;
  actualDuration: number;
  baseDuration: number;
}

let logQueue: PerfLogObject[] = [];

const now = (): string =>
  new Date().toLocaleTimeString('en-US', {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
  });

const log = (debug, ...args): void => {
  if (typeof debug !== 'boolean') {
    args.unshift(debug);
    debug = false;
  }
  const msg = [
    '###PERF###',
    now(),
    ...args.map(arg => (typeof arg === 'number' ? arg.toFixed(3) : arg)),
  ];
  const _log = debug === true ? console.debug : console.log;
  _log(...msg);
};

export const measureInteraction = (interactionName: string): {end: () => void} => {
  // performance.now() returns the number of ms
  // elapsed since the page was opened
  const startTimestamp = performance.now();

  return {
    end() {
      const endTimestamp = performance.now();
      log(`${interactionName} interaction took`, endTimestamp - startTimestamp, 'ms');
    },
  };
};

export const markInteraction = (interactionName: string): void => {
  const interaction = measureInteraction(interactionName);
  afterFrame(() => {
    interaction.end();
  });
};

export const logRender: ProfilerOnRenderCallback = (
  id,
  phase,
  actualDuration,
  baseDuration,
  _startTime,
  _commitTime,
  _interactions
) => {
  logQueue.push({
    id,
    phase,
    actualDuration,
    baseDuration,
  });
  log(true, 'Rendered', id, phase, 'act:', actualDuration, 'ms', ' base:', baseDuration, 'ms');
};

export const logAggregation = (): void => {
  const data = logQueue.reduce((acc: {[key: string]: PerfLogObject}, curr) => {
    const {id, phase, actualDuration, baseDuration} = curr;
    const key = `${id}-${phase}`;
    if (acc[key] != null) {
      acc[key].actualDuration += actualDuration;
      acc[key].baseDuration += baseDuration; // Is it ok to sum the baseDuration?
    } else {
      acc[key] = {
        id,
        phase,
        actualDuration,
        baseDuration,
      };
    }
    return acc;
  }, {});

  Object.keys(data)
    .sort()
    .forEach(key => {
      const {id, phase, actualDuration, baseDuration} = data[key];
      log(
        'Aggregate(last 1s) rendered',
        id,
        phase,
        'act:',
        actualDuration,
        'ms',
        ' base:',
        baseDuration,
        'ms'
      );
    });
  logQueue = [];
};

setInterval(() => {
  logAggregation();
}, 1000);
