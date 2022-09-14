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

import {createContext, ReactNode, useContext} from 'react';
import Spinner from '../Spinner';

interface ParcaThemeContextProps {
  loader: ReactNode;
}

const defaultValue: ParcaThemeContextProps = {
  loader: <Spinner />,
};

const ParcaThemeContext = createContext<ParcaThemeContextProps>(defaultValue);

export const ParcaThemeProvider = ({
  children,
  value,
}: {
  children: ReactNode;
  value?: ParcaThemeContextProps;
}): JSX.Element => {
  return (
    <ParcaThemeContext.Provider value={value ?? defaultValue}>
      {children}
    </ParcaThemeContext.Provider>
  );
};

export const useParcaTheme = (): ParcaThemeContextProps => {
  const context = useContext(ParcaThemeContext);
  if (context == null) {
    return defaultValue;
  }
  return context;
};

export default ParcaThemeContext;
