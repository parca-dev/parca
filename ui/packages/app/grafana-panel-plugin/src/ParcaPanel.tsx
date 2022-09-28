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

import React, { useState } from 'react';
import { Provider } from 'react-redux';
import { PanelProps } from '@grafana/data';
import { css, cx } from 'emotion';
import { stylesFactory, useTheme } from '@grafana/ui';
import { ProfileView, VisualizationType, ProfileVisState } from '@parca/profile';
import { store } from '@parca/store';

import '@parca/profile/dist/styles.css';
import '@parca/components/dist/styles.css';

interface Props extends PanelProps<{}> {}

const { store: parcaStore } = store();

const useInMemoryProfileVisState = (): ProfileVisState => {
  const [currentView, setCurrentView] = useState<VisualizationType>('icicle');

  return { currentView, setCurrentView };
};

export const ParcaPanel: React.FC<Props> = ({ data, width, height }) => {
  const theme = useTheme();
  const styles = getStyles();

  const profileVisState = useInMemoryProfileVisState();

  const { flamegraphData, topTableData } = data.series[0]?.fields[0].values.get(0) ?? {};

  return (
    <Provider store={parcaStore}>
      <div
        className={cx(
          styles.wrapper,
          css`
            width: ${width}px;
            height: ${height}px;
          `,
          { dark: theme.isDark }
        )}
      >
        <ProfileView
          flamegraphData={flamegraphData}
          topTableData={topTableData}
          sampleUnit={flamegraphData?.unit ?? 'bytes'}
          onDownloadPProf={() => {}}
          profileVisState={profileVisState}
        />
      </div>
    </Provider>
  );
};

const getStyles = stylesFactory(() => {
  return {
    wrapper: css`
      position: relative;
      overflow: hidden;
      z-index: 0;
    `,
    svg: css`
      position: absolute;
      top: 0;
      left: 0;
    `,
    textBox: css`
      position: absolute;
      bottom: 0;
      left: 0;
      padding: 10px;
    `,
  };
});
