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

import {Profiler, ProfilerOnRenderCallback} from 'react';

import {QueryServiceClient} from '@parca/client';
import {ConditionalWrapper} from '@parca/components';

import ProfileFlameGraph from '../../../ProfileFlameGraph';
import {CurrentPathFrame} from '../../../ProfileFlameGraph/FlameGraphArrow/utils';
import {ProfileSource} from '../../../ProfileSource';
import Sandwich from '../../../Sandwich';
import {SourceView} from '../../../SourceView';
import {Table} from '../../../Table';
import type {
  FlamegraphData,
  SandwichData,
  SourceData,
  TopTableData,
  VisualizationType,
} from '../../types/visualization';

interface GetDashboardItemProps {
  type: VisualizationType;
  isHalfScreen: boolean;
  dimensions: DOMRect | undefined;
  flamegraphData: FlamegraphData;
  flamechartData: FlamegraphData;
  topTableData?: TopTableData;
  sandwichData: SandwichData;
  sourceData?: SourceData;
  profileSource: ProfileSource;
  total: bigint;
  filtered: bigint;
  curPath: string[];
  setNewCurPath: (path: string[]) => void;
  curPathArrow: CurrentPathFrame[];
  setNewCurPathArrow: (path: CurrentPathFrame[]) => void;
  perf?: {
    onRender?: ProfilerOnRenderCallback;
  };
  queryClient?: QueryServiceClient;
}

export const getDashboardItem = ({
  type,
  isHalfScreen,
  dimensions,
  flamegraphData,
  flamechartData,
  topTableData,
  sourceData,
  sandwichData,
  profileSource,
  total,
  filtered,
  curPathArrow,
  setNewCurPathArrow,
  perf,
}: GetDashboardItemProps): JSX.Element => {
  switch (type) {
    case 'flamegraph':
      return (
        <ConditionalWrapper
          condition={perf?.onRender != null}
          WrapperComponent={Profiler}
          wrapperProps={{
            id: 'flameGraph',
            onRender: perf?.onRender ?? (() => {}),
          }}
        >
          <ProfileFlameGraph
            curPathArrow={curPathArrow}
            setNewCurPathArrow={setNewCurPathArrow}
            arrow={flamegraphData?.arrow}
            total={total}
            filtered={filtered}
            profileType={profileSource?.ProfileType()}
            loading={flamegraphData.loading}
            error={flamegraphData.error}
            isHalfScreen={isHalfScreen}
            width={
              dimensions?.width !== undefined
                ? isHalfScreen
                  ? (dimensions.width - 54) / 2
                  : dimensions.width - 16
                : 0
            }
            metadataMappingFiles={flamegraphData.metadataMappingFiles}
            metadataLoading={flamegraphData.metadataLoading}
            profileSource={profileSource}
          />
        </ConditionalWrapper>
      );
    case 'flamechart':
      return (
        <ProfileFlameGraph
          curPathArrow={[]}
          setNewCurPathArrow={() => {}}
          arrow={flamechartData?.arrow}
          total={total}
          filtered={filtered}
          profileType={profileSource?.ProfileType()}
          loading={flamechartData.loading}
          error={flamechartData.error}
          isHalfScreen={isHalfScreen}
          width={
            dimensions?.width !== undefined
              ? isHalfScreen
                ? (dimensions.width - 54) / 2
                : dimensions.width - 16
              : 0
          }
          metadataMappingFiles={flamechartData.metadataMappingFiles}
          metadataLoading={flamechartData.metadataLoading}
          profileSource={profileSource}
          isFlameChart={true}
        />
      );
    case 'table':
      return topTableData != null ? (
        <Table
          total={total}
          filtered={filtered}
          loading={topTableData.loading}
          data={topTableData.arrow?.record}
          unit={topTableData.unit}
          profileType={profileSource?.ProfileType()}
          isHalfScreen={isHalfScreen}
          metadataMappingFiles={flamegraphData.metadataMappingFiles}
        />
      ) : (
        <></>
      );
    case 'sandwich':
      return topTableData != null ? (
        <Sandwich profileSource={profileSource} sandwichData={sandwichData} />
      ) : (
        <></>
      );
    case 'source':
      return sourceData != null ? (
        <SourceView
          loading={sourceData.loading}
          data={sourceData.data}
          total={total}
          filtered={filtered}
        />
      ) : (
        <></>
      );
    default:
      return <></>;
  }
};
