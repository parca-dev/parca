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

import {useState} from 'react';

import {Icon} from '@iconify/react';
import cx from 'classnames';

import {USER_PREFERENCES} from '@parca/hooks';

import {Button} from '../Button';
import Modal from '../Modal';
import FlamegraphColorProfileSelector from './FlamegraphColorProfileSelector';
import UserPreferenceItem from './UserPreferenceItem';

export const UserPreferencesModal = (): JSX.Element => {
  const [isOpen, setIsOpen] = useState<boolean>(false);

  return (
    <>
      <Button
        onClick={() => setIsOpen(true)}
        className="gap-2"
        variant="neutral"
        id="h-viz-preferences"
      >
        Preferences
        <Icon icon="pajamas:preferences" width={20} />
      </Button>

      {isOpen && (
        <Modal
          isOpen={isOpen}
          closeModal={() => {
            setIsOpen(false);
          }}
          title="Visualisation Preferences"
          className="max-w-[600px]"
        >
          <UserPreferences modal={true} />
        </Modal>
      )}
    </>
  );
};

const UserPreferences = ({modal}: {modal?: boolean}): JSX.Element => {
  return (
    <div>
      <div className={cx('min-h-40 ', modal === true ? '' : 'mt-8')}>
        <UserPreferenceItem
          id="h-highlight-after-filtering"
          userPreferenceDetails={USER_PREFERENCES.HIGHTLIGHT_AFTER_FILTERING}
        />
        <UserPreferenceItem
          id="h-dock-graph-tooltip"
          userPreferenceDetails={USER_PREFERENCES.GRAPH_METAINFO_DOCKED}
        />
        <UserPreferenceItem
          id="h-highlight-similar-stacks"
          userPreferenceDetails={USER_PREFERENCES.HIGHLIGHT_SIMILAR_STACKS}
        />
        <UserPreferenceItem
          id="h-show-function-name-from-left"
          userPreferenceDetails={USER_PREFERENCES.SHOW_FUNCTION_NAME_FROM_LEFT}
        />
        <UserPreferenceItem
          id="h-enable-iciclechart"
          userPreferenceDetails={USER_PREFERENCES.ENABLE_ICICLECHARTS}
        />
        <FlamegraphColorProfileSelector />
      </div>
    </div>
  );
};

export default UserPreferences;
