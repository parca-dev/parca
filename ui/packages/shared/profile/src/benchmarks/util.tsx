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

import * as React from 'react';
import {Benchmark} from 'react-component-benchmark';
import type {BenchmarkType, BenchmarkRef, BenchResultsType} from 'react-component-benchmark';
import {act, render, screen, waitFor} from '@testing-library/react';
import {Provider} from 'react-redux';
import {store} from '@parca/store';

interface Props {
  component: React.ComponentType;
  props?: Record<string, unknown>;
  samples?: number;
  type?: BenchmarkType;
}

const setupMocks = (): void => {
  window.ResizeObserver =
    window.ResizeObserver ??
    jest.fn().mockImplementation(() => ({
      disconnect: jest.fn(),
      observe: jest.fn(),
      unobserve: jest.fn(),
    }));

  window.HTMLElement.prototype.getBoundingClientRect = function () {
    // Very hacky mock, could be the first reason for any failures related to BoundingClientRect
    const marginTop = this.style.marginTop !== '' ? parseFloat(this.style.marginTop) : 0;
    const marginLeft = this.style.marginLeft !== '' ? parseFloat(this.style.marginLeft) : 0;
    const width = this.style.width !== '' ? parseFloat(this.style.width) : 0;
    const height = this.style.height !== '' ? parseFloat(this.style.height) : 0;
    if (marginTop === 0 && marginLeft === 0 && width === 0 && height === 0) {
      return this.parentElement?.getBoundingClientRect();
    }
    return {
      width,
      height,
      top: marginTop,
      left: marginLeft,
      x: marginLeft,
      y: marginTop,
      right: width,
      bottom: height,
      toJSON: () => {},
    };
  };
};

/**
 * A wrapper function to make benchmarking in tests a bit more reusable.
 * You might tune this to your specific needs
 * @param  {React.Component} options.component  The component you'd like to benchmark
 * @param  {Object} options.props               Props for your component
 * @param  {Number} options.samples             Number of samples to take. default 50 is a safe number
 * @param  {String} options.type                Lifecycle of a component ('mount', 'update', or 'unmount')
 * @return {Object}                             Results object
 */
export async function runBenchmark({
  component,
  props,
  samples = 25,
  type = 'mount',
}: Props): Promise<BenchResultsType> {
  // Benchmarking requires a real time system and not mocks. Ensure you're not using fake timers
  jest.useRealTimers();
  setupMocks();

  const ref = React.createRef<BenchmarkRef>();

  let results: BenchResultsType;
  const handleComplete = jest.fn(res => {
    results = res;
  });
  const {store: reduxStore} = store();

  render(
    <Provider store={reduxStore}>
      <div style={{width: 2536, height: 1315}}>
        <Benchmark
          component={component}
          onComplete={handleComplete}
          ref={ref}
          samples={samples}
          componentProps={props}
          type={type}
        />
      </div>
    </Provider>
  );

  act(() => {
    ref.current?.start();
  });

  await waitFor(() => expect(handleComplete).toHaveBeenCalled(), {timeout: 10000});
  //screen.debug();
  // @ts-expect-error
  return results;
}
