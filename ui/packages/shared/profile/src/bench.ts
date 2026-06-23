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

type BenchWindow = Window &
  typeof globalThis & {
    __bench?: boolean;
    __benchMeta?: Record<string, number>;
  };

let queryBenchEnabled: boolean | undefined;

export const benchEnabled = (): boolean => {
  if (typeof window === 'undefined') return false;

  if (queryBenchEnabled === undefined) {
    queryBenchEnabled = new URLSearchParams(window.location.search).get('bench') === '1';
  }
  return queryBenchEnabled || (window as BenchWindow).__bench === true;
};

export const mark = (name: string): void => {
  if (!benchEnabled()) return;
  performance.mark(name);
};

export const measure = (name: string, start: string, end: string): void => {
  if (!benchEnabled()) return;
  try {
    performance.measure(name, start, end);
  } catch {
    // A missing mark should not affect the profiler page.
  }
};

export const afterPaint = (cb: () => void): void => {
  if (!benchEnabled()) return;
  requestAnimationFrame(() => requestAnimationFrame(cb));
};

// Stash a non-timing value (e.g. node count) for the driver to read off window.__benchMeta.
export const benchMeta = (key: string, value: number): void => {
  if (!benchEnabled()) return;
  const win = window as BenchWindow;
  (win.__benchMeta ??= {})[key] = value;
};
