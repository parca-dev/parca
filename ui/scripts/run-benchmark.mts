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

import path from 'path';
import {fileURLToPath} from 'url';

import commandLineArgs from 'command-line-args';
import {execa} from 'execa';
import fs from 'fs-extra';
import glob from 'glob-promise';
import notLog from 'not-a-log';
import ora from 'ora';

import ReactBenchmark from '@parca/react-benchmark';

import StopWatch from './stop-watch.js';

const DIR_NAME = path.dirname(fileURLToPath(import.meta.url));

const spinner = ora();

const optionDefinitions = [
  {name: 'name', alias: 'n', type: String},
  {name: 'debug', alias: 'd', type: Boolean},
  {name: 'compare', alias: 'c', type: String},
  {name: 'pattern', alias: 'p', type: String, defaultValue: '*'},
];
const options = commandLineArgs(optionDefinitions);
const IS_DEBUG = process.env.DEBUG === 'true' || options.debug === true;

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

class ComparedValue {
  before: number;
  after: number;
  constructor(before: number, after: number) {
    this.before = before;
    this.after = after;
  }

  toString() {
    const diff = this.after - this.before;
    const diffPercentage = (diff / this.before) * 100;
    const diffSign = diff > 0 ? '+' : '';
    return `${this.after} (${diffSign}${diff.toFixed(2)}, ${diffSign}${diffPercentage.toFixed(
      2
    )}%)`;
  }
}

interface CompareResult {
  name: string;
  mean: string;
  min: string;
  max: string;
  p50: string;
  p75: string;
  p90: string;
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

  if (options.name != null) {
    const reportPath = path.join(
      DIR_NAME,
      `../perf-benchmark-reports/${options.name as string}.txt`
    );
    await fs.ensureFile(reportPath);
    await fs.appendFile(reportPath, report);
    const jsonReportPath = path.join(
      DIR_NAME,
      `../perf-benchmark-reports/${options.name as string}.json`
    );
    await fs.ensureFile(jsonReportPath);
    await fs.writeFile(jsonReportPath, JSON.stringify({benchmarks: results}, null, 2));
  }

  if (options.compare != null) {
    const compareFile = path.resolve(DIR_NAME, `../perf-benchmark-reports/${options.compare}.json`);
    const compareReport = await fs.readJSON(compareFile, 'utf-8');
    const beforeResults = compareReport.benchmarks as Result[];
    const comparedResults = beforeResults
      .map(beforeResult => {
        const result = results.find(r => r.name === beforeResult.name);
        if (result === undefined) {
          return;
        }
        const compareResult: CompareResult = {
          name: result.name,
          mean: new ComparedValue(beforeResult.mean, result.mean).toString(),
          min: new ComparedValue(beforeResult.min, result.min).toString(),
          max: new ComparedValue(beforeResult.max, result.max).toString(),
          p50: new ComparedValue(beforeResult.p50, result.p50).toString(),
          p75: new ComparedValue(beforeResult.p75, result.p75).toString(),
          p90: new ComparedValue(beforeResult.p90, result.p90).toString(),
        };
        return compareResult;
      })
      .filter(Boolean);
    const compareTable = notLog.table(comparedResults, [
      'name',
      'mean',
      'min',
      'max',
      'p50',
      'p75',
      'p90',
    ]);
    console.log(`Compare with ${options.compare}:`);
    console.log(compareTable);
  }
};

const populateBenchmarkData = async (): Promise<void> => {
  const stopwatch = new StopWatch();
  stopwatch.start();
  spinner.start('Discovering data population scripts');
  const files = await glob('!(node_modules)/**/!(node_modules|dist)/**/benchdata/populateData.js');
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
  const files = await glob(`!(node_modules)/**/!(node_modules)/${options.pattern}.benchmark.tsx`);
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
