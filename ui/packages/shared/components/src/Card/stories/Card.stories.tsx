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

import TextareaAutosize from 'react-textarea-autosize';

import Card from '..';
import {Button} from '../../Button';
import {NoDataPrompt} from '../../NoDataPrompt';

const ComponentStory = (): JSX.Element => (
  <div className="ml-8">
    <Card>
      <Card.Header className="flex space-x-2 !items-center">
        <div className="flex flex-wrap w-full justify-start space-x-2 space-y-1 items-center">
          <div className="ml-2 mt-1">Parca</div>

          <div className="w-full flex-1">
            <TextareaAutosize
              className="bg-gray-50 dark:bg-gray-900 focus:ring-indigo-800 flex-1 block w-full px-2 py-2 text-sm outline-none rounded"
              placeholder="Select a profile first to enter a filter..."
              title="Select a profile first to enter a filter..."
            />
          </div>

          <div>
            <Button>Search</Button>
          </div>
        </div>
      </Card.Header>
      <Card.Body>
        <NoDataPrompt />
      </Card.Body>
    </Card>
  </div>
);

export default {
  title: 'Components/Card ',
  component: ComponentStory,
};

export const Default = ComponentStory.bind({});
