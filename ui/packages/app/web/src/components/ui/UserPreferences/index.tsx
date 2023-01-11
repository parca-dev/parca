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
import {Button, IconButton, Modal} from '@parca/components';
import useUIFeatureFlag from '@parca/functions/useUIFeatureFlag';
import {USER_PREFERENCES} from '@parca/functions/useUserPreference';
import {useState} from 'react';
import UserPreferenceItem from './UserPreferenceItem';

interface FlagToggleProps {
  name: string;
  id: string;
}

const FlagToggle = ({name, id}: FlagToggleProps): JSX.Element => {
  const [enabled, setEnabled] = useUIFeatureFlag(id);

  return (
    <div className="flex gap-2">
      <input
        type="checkbox"
        id={id}
        checked={enabled}
        onChange={e => setEnabled(e.target.checked)}
      />
      <label htmlFor={id}>{name}</label>
    </div>
  );
};

const UserPreferences = () => {
  const [isOpen, setIsOpen] = useState<boolean>(false);
  return (
    <div>
      <IconButton onClick={() => setIsOpen(!isOpen)}>
        <Icon icon="material-symbols:settings-outline-rounded" fontSize={20} />
      </IconButton>
      <Modal
        isOpen={isOpen}
        closeModal={() => {
          setIsOpen(false);
        }}
        title="Preferences"
        className="max-w-[460px]"
      >
        <div className="min-h-40 mt-8">
          <UserPreferenceItem userPreferenceDetails={USER_PREFERENCES.HIGHTLIGHT_AFTER_FILTERING} />
          <UserPreferenceItem userPreferenceDetails={USER_PREFERENCES.DISABLE_GRAPH_TRIMMING} />
          <div className="min-w-96 mt-10">
            <h4 className="font-medium mb-2">Experimental Features</h4>
            <FlagToggle name="Enable Callgraph" id="callgraph" />
          </div>
        </div>
        <div className="flex justify-end">
          <Button onClick={() => setIsOpen(false)} className="w-fit">
            Close
          </Button>
        </div>
      </Modal>
    </div>
  );
};

export default UserPreferences;
