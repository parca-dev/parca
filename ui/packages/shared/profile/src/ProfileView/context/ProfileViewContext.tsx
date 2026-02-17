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

import {ReactNode, createContext, useContext} from 'react';

import {Table} from '@uwdata/flechette';

import {ProfileSource} from '../../ProfileSource';
import {NumberDuo} from '../../utils';

export type TimelineGuideData =
  | {show: false}
  | {
      show: true;
      props: {
        bounds: NumberDuo;
      };
    };

interface Props {
  profileSource?: ProfileSource;
  compareMode: boolean;
  timelineGuide?: TimelineGuideData;
  metadataMappingFiles?: string[];
  flamegraphTable?: Table | null;
}

export const defaultValue: Props = {
  profileSource: undefined,
  compareMode: false,
  timelineGuide: {show: false},
};

const ProfileViewContext = createContext<Props>(defaultValue);

export const ProfileViewContextProvider = ({
  children,
  value,
}: {
  children: ReactNode;
  value?: Props;
}): JSX.Element => {
  return (
    <ProfileViewContext.Provider value={{...defaultValue, ...(value ?? {})}}>
      {children}
    </ProfileViewContext.Provider>
  );
};

export const useProfileViewContext = (): Props => {
  const context = useContext(ProfileViewContext);
  if (context == null) {
    return defaultValue;
  }
  return context;
};

export default ProfileViewContext;
