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

import {Icon} from '@iconify/react';

const MetricsInfoPanel = ({isInfoPanelOpen, onInfoIconClick}): JSX.Element => {
  return (
    <div>
      {isInfoPanelOpen ? (
        <div className="flex flex-col items-end gap-1">
          <Icon
            icon="material-symbols:info"
            width={25}
            height={25}
            className="cursor-pointer text-gray-500"
          />
          <div className="shadow-allSideOuter h-[300px] w-[400px] rounded-md border bg-gray-100">
            <div className="flex w-full items-center gap-1">
              <Icon icon="iconoir:mouse-button-left" />
              <div>Placeholder text</div>
            </div>
          </div>
        </div>
      ) : (
        <Icon
          icon="material-symbols:info-outline"
          width={25}
          height={25}
          onClick={onInfoIconClick}
          className="cursor-pointer text-gray-500"
        />
      )}
    </div>
  );
};

export default MetricsInfoPanel;
