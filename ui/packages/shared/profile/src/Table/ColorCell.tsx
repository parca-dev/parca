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

interface ColorCellProps {
  color: string;
  mappingFile: string;
}

export const ColorCell = ({color, mappingFile}: ColorCellProps): JSX.Element => (
  <div
    className="w-4 h-4 rounded-[4px]"
    style={{backgroundColor: color}}
    data-tooltip-id="table-color-tooltip"
    data-tooltip-content={mappingFile}
  />
);
