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

class StopWatch {
  private _startTime: number;
  private _endTime: number;
  private _duration: number;

  constructor() {
    this._startTime = 0;
    this._endTime = 0;
    this._duration = 0;
  }

  start(): void {
    this._startTime = new Date().getTime();
  }

  stop(): number {
    this._endTime = new Date().getTime();
    this._duration = this._endTime - this._startTime;
    return this.duration;
  }

  get duration(): number {
    return this._duration;
  }

  reset(): void {
    this._startTime = 0;
    this._endTime = 0;
    this._duration = 0;
  }

  stopAndReset(): number {
    const duration = this.stop();
    this.reset();
    return duration;
  }
}

export default StopWatch;
