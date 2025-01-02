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

import {ConditionalWrapper} from '@parca/components';

import Callgraph from '../../../Callgraph';
import ProfileIcicleGraph from '../../../ProfileIcicleGraph';
import {ProfileSource} from '../../../ProfileSource';
import {SourceView} from '../../../SourceView';
import {Table} from '../../../Table';
import type {
  CallgraphData,
  FlamegraphData,
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
  callgraphData?: CallgraphData;
  sourceData?: SourceData;
  profileSource?: ProfileSource;
  total: bigint;
  filtered: bigint;
  curPath: string[];
  setNewCurPath: (path: string[]) => void;
  currentSearchString?: string;
  setSearchString?: (value: string) => void;
  callgraphSVG?: string;
  perf?: {
    onRender?: ProfilerOnRenderCallback;
  };
}

export const getDashboardItem = ({
  type,
  isHalfScreen,
  dimensions,
  flamegraphData,
  flamechartData,
  topTableData,
  callgraphData,
  sourceData,
  profileSource,
  total,
  filtered,
  curPath,
  setNewCurPath,
  currentSearchString,
  setSearchString,
  callgraphSVG,
  perf,
}: GetDashboardItemProps): JSX.Element => {
  switch (type) {
    case 'icicle':
      return (
        <ConditionalWrapper
          condition={perf?.onRender != null}
          WrapperComponent={Profiler}
          wrapperProps={{
            id: 'icicleGraph',
            onRender: perf?.onRender ?? (() => {}),
          }}
        >
          <ProfileIcicleGraph
            curPath={curPath}
            setNewCurPath={setNewCurPath}
            arrow={flamegraphData?.arrow}
            graph={flamegraphData?.data}
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
          />
        </ConditionalWrapper>
      );
    case 'iciclechart':
      return (
        <ProfileIcicleGraph
          curPath={curPath}
          setNewCurPath={setNewCurPath}
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
          isIcicleChart={true}
        />
      );
    case 'callgraph':
      return callgraphData?.data !== undefined &&
        callgraphSVG !== undefined &&
        dimensions?.width !== undefined ? (
        <Callgraph
          data={callgraphData.data}
          svgString={callgraphSVG}
          profileType={profileSource?.ProfileType()}
          width={isHalfScreen ? dimensions?.width / 2 : dimensions?.width}
        />
      ) : (
        <></>
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
          currentSearchString={currentSearchString}
          setSearchString={setSearchString}
          isHalfScreen={isHalfScreen}
          metadataMappingFiles={flamegraphData.metadataMappingFiles}
        />
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
