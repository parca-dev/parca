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

import {MoonIcon, SunIcon} from '@heroicons/react/solid';
import {useAppSelector, useAppDispatch, setDarkMode, selectDarkMode} from '@parca/store';

const DarkModeToggle = () => {
  const dispatch = useAppDispatch();
  const isDarkMode = useAppSelector(selectDarkMode);
  const Icon = isDarkMode ? MoonIcon : SunIcon;
  return (
    <button
      type="button"
      className="p-1 cursor-pointer rounded-full flex align-center items-center"
      onClick={() => {}}
    >
      <Icon
        onClick={() => dispatch(setDarkMode(!isDarkMode))}
        className="w-5 h-5"
        aria-hidden="true"
      />
    </button>
  );
};

export default DarkModeToggle;
