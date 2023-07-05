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

import React from 'react';
import { Provider } from 'react-redux';
import { PanelProps } from '@grafana/data';
import { css, cx } from 'emotion';
import { stylesFactory } from '@grafana/ui';
import { ProfileView, GrafanaParcaData, MergedProfileSource } from '@parca/profile';
import { store } from '@parca/store';

import '@parca/profile/dist/styles.css';
import '@parca/components/dist/styles.css';

interface Props extends PanelProps<{}> {}

const { store: parcaStore } = store();

function extractData<T>(data: any): T {
  return data.series[0].fields[0].values.get(0);
}

export const ParcaPanel: React.FC<Props> = ({ data, width, height }) => {
  const styles = getStyles();

  const response = extractData<GrafanaParcaData>(data);

  if (response.error !== undefined) {
    console.error('Error loading profile', response.error);
    console.log('response.error', response.error);
    return (
      <div className={styles.errorWrapper}>
        <span>Something went wrong!</span>
        <span>{response.error?.message}</span>
      </div>
    );
  }

  const { flamegraphData, topTableData, actions } = response;

  let total = BigInt(0);
  if (flamegraphData.total !== undefined) {
    total = BigInt(flamegraphData.total);
  } else if (topTableData.total !== undefined) {
    total = BigInt(topTableData.total);
  }

  return (
    <Provider store={parcaStore}>
      <div
        className={cx(
          styles.wrapper,
          css`
            width: ${width}px;
            height: ${height}px;
          `
        )}
      >
        <ProfileView
          total={total}
          filtered={BigInt(0)}
          flamegraphData={flamegraphData}
          topTableData={topTableData}
          sampleUnit={flamegraphData.data?.unit ?? 'bytes'}
          onDownloadPProf={actions.downloadPprof}
          profileSource={
            new MergedProfileSource(
              data.timeRange.from.valueOf(),
              data.timeRange.to.valueOf(),
              (data.request?.targets[0] as any).parcaQuery,
            )
          }
          queryClient={actions.getQueryClient()}
        />
      </div>
    </Provider>
  );
};

const getStyles = stylesFactory(() => {
  return {
    wrapper: css`
      position: relative;
      overflow: scroll;
      z-index: 0;
    `,
    svg: css`
      position: absolute;
      top: 0;
      left: 0;
    `,
    errorWrapper: css`
      display: flex;
      justify-content: center;
      align-items: center;
      width: 100%;
      height: 100%;
      flex-direction: column;
      text-align: center;
    `,
  };
});
