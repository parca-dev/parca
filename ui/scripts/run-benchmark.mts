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

import glob from 'glob-promise';
import path from 'path';
import {fileURLToPath} from 'url';
import ReactBenchmark from '@parca/react-benchmark';
import StopWatch from './stop-watch.js';
import ora from 'ora';
import commandLineArgs from 'command-line-args';
import notLog from 'not-a-log';
import {execa} from 'execa';
import fs from 'fs-extra';

const DIR_NAME = path.dirname(fileURLToPath(import.meta.url));
const IS_DEBUG = process.env.DEBUG === 'true';

const spinner = ora();

const optionDefinitions = [{name: 'name', alias: 'n', type: String}];
const options = commandLineArgs(optionDefinitions);

interface Result {
  name: string;
  mean: number;
  sampleCount: number;
  samples: number[];
  variance: number;
  hz: number;
  min: number;
  max: number;
  p50: number;
  p75: number;
  p90: number;
}

const runTime = new Date();

const saveReport = async (results: Result[]): Promise<void> => {
  const table = notLog.table(results, [
    'name',
    'sampleCount',
    'mean',
    'min',
    'max',
    'p50',
    'p75',
    'p90',
  ]);
  const report = `############## Benchmark Report ##############
Run at: ${runTime.toLocaleString()}
Results:
${table}

`;

  console.log(report);

  if (options.name === undefined || options.name == null) {
    return;
  }
  const reportPath = path.join(DIR_NAME, `../perf-benchmark-reports/${options.name as string}.txt`);
  await fs.ensureFile(reportPath);
  await fs.appendFile(reportPath, report);
};

const populateBenchmarkData = async (): Promise<void> => {
  const stopwatch = new StopWatch();
  stopwatch.start();
  spinner.start('Discovering data population scripts');
  const files = await glob('!(node_modules)/**/!(node_modules|dist)/benchdata/populateData.js');
  spinner.succeed(`Found ${files.length} data population scripts ${stopwatch.stopAndReset()}ms`);
  for (const file of files) {
    try {
      stopwatch.start();
      spinner.start(`Running data population script: ${file}`);
      const {stdout} = await execa('babel-node', [
        '--config-file',
        path.join(DIR_NAME, '../babel.config.cjs'),
        path.join(DIR_NAME, `../${file}`),
      ]);
      console.log('stdout', stdout);
      spinner.succeed(`Data population script: ${file} completed ${stopwatch.stopAndReset()}ms`);
    } catch (error) {
      spinner.fail(`Data population script: ${file} failed ${stopwatch.stopAndReset()}ms`);
      throw error;
    }
  }
};

const run = async (): Promise<void> => {
  spinner.start('Discovering benchmarks');
  const files = await glob('!(node_modules)/**/!(node_modules)/*.benchmark.tsx');
  spinner.succeed(`Found ${files.length} benchmarks`);

  await populateBenchmarkData();

  const results: Result[] = [];

  for (const file of files) {
    try {
      const result = await runBenchmark(file).catch(() => {
        return null;
      });
      if (result == null) {
        continue;
      }
      results.push(result);
    } catch (error) {
      console.log('Error running benchmark', file, error);
    }
  }

  await saveReport(results);
};

const runBenchmark = async (file: string): Promise<Result> => {
  const stopwatch = new StopWatch();
  const benchmarkName = file.split(path.sep).slice(-1)[0].replace('.benchmark.tsx', '');
  try {
    const reactBenchmark = new ReactBenchmark();
    if (IS_DEBUG) {
      reactBenchmark.on('console', message => console.log('console', message));
    }
    reactBenchmark.on('webpack', () => {
      stopwatch.start();
      spinner.start(`${benchmarkName} ::: Webpack compiling`);
    });
    reactBenchmark.on('server', () => {
      spinner.succeed(`${benchmarkName} ::: Webpack compiling ${stopwatch.stopAndReset()}ms`);
      stopwatch.start();
      spinner.start(`${benchmarkName} ::: Starting server`);
    });
    reactBenchmark.on('chrome', () => {
      spinner.succeed(`${benchmarkName} ::: Starting server ${stopwatch.stopAndReset()}ms`);
      stopwatch.start();
      spinner.start(`${benchmarkName} ::: Starting chrome`);
    });
    reactBenchmark.on('start', () => {
      spinner.succeed(`${benchmarkName} ::: Starting chrome ${stopwatch.stopAndReset()}ms`);
      stopwatch.start();
      spinner.start(`${benchmarkName} ::: Running benchmark`);
    });
    const result = await reactBenchmark.run(file, {debug: IS_DEBUG, devtools: IS_DEBUG});
    spinner.succeed(`${benchmarkName} ::: Running benchmark ${stopwatch.stopAndReset()}ms`);
    const sortedSamples = result.stats.sample.sort((a, b) => a - b);

    return {
      name: benchmarkName,
      mean: toMS(result.stats.mean),
      sampleCount: result.stats.sample.length,
      samples: result.stats.sample,
      variance: result.stats.variance,
      hz: result.hz,
      min: toMS(sortedSamples[0]), // samples are sorted already
      max: toMS(sortedSamples[sortedSamples.length - 1]),
      p50: toMS(sortedSamples[Math.floor(sortedSamples.length * 0.5)]),
      p75: toMS(sortedSamples[Math.floor(sortedSamples.length * 0.75)]),
      p90: toMS(sortedSamples[Math.floor(sortedSamples.length * 0.9)]),
    };
  } catch (error) {
    spinner.fail(`${benchmarkName} ::: Running benchmark ${stopwatch.stopAndReset()}ms`);
    console.error('Error running benchmark', file, error);
    throw error;
  }
};

const toMS = (sec: number): number => {
  return parseFloat((sec * 1000).toFixed(3));
};

run().catch(error => {
  console.error('Error running benchmarks', error);
  process.exit(1);
});
