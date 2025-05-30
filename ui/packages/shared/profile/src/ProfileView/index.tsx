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

import {KeyDownProvider, useParcaContext} from '@parca/components';
import {useContainerDimensions} from '@parca/hooks';
import {selectQueryParam} from '@parca/utilities';

import ColorStackLegend from './components/ColorStackLegend';
import {getDashboardItem} from './components/DashboardItems';
import {DashboardLayout} from './components/DashboardLayout';
import {ProfileHeader} from './components/ProfileHeader';
import {IcicleGraphToolbar, TableToolbar, VisualisationToolbar} from './components/Toolbars';
import {DashboardProvider} from './context/DashboardContext';
import {ProfileViewContextProvider} from './context/ProfileViewContext';
import {useProfileMetadata} from './hooks/useProfileMetadata';
import {useVisualizationState} from './hooks/useVisualizationState';
import type {ProfileViewProps, VisualizationType} from './types/visualization';

export const ProfileView = ({
  total,
  filtered,
  flamegraphData,
  flamechartData,
  topTableData,
  sourceData,
  profileSource,
  queryClient,
  onDownloadPProf,
  pprofDownloading,
  compare,
  showVisualizationSelector,
}: ProfileViewProps): JSX.Element => {
  const {
    timezone,
    perf,
    profileViewExternalMainActions,
    preferencesModal,
    profileViewExternalSubActions,
  } = useParcaContext();
  const {ref, dimensions} = useContainerDimensions();

  const {
    curPath,
    setCurPath,
    curPathArrow,
    setCurPathArrow,
    currentSearchString,
    setSearchString,
    colorStackLegend,
    colorBy,
    groupBy,
    toggleGroupBy,
    clearSelection,
    setGroupByLabels,
  } = useVisualizationState();

  const {colorMappings} = useProfileMetadata({
    flamegraphArrow: flamegraphData.arrow,
    metadataMappingFiles: flamegraphData.metadataMappingFiles,
    metadataLoading: flamegraphData.metadataLoading,
    colorBy,
  });

  const isColorStackLegendEnabled = colorStackLegend === 'true';
  const compareMode =
    compare === true ||
    (selectQueryParam('compare_a') === 'true' && selectQueryParam('compare_b') === 'true');

  const getDashboardItemByType = ({
    type,
    isHalfScreen,
  }: {
    type: VisualizationType;
    isHalfScreen: boolean;
  }): JSX.Element => {
    return getDashboardItem({
      type,
      isHalfScreen,
      dimensions,
      flamegraphData,
      flamechartData,
      topTableData,
      sourceData,
      profileSource,
      total,
      filtered,
      curPath,
      setNewCurPath: setCurPath,
      curPathArrow,
      setNewCurPathArrow: setCurPathArrow,
      currentSearchString,
      setSearchString,
      perf,
    });
  };

  const actionButtons = {
    icicle: <IcicleGraphToolbar curPath={curPathArrow} setNewCurPath={setCurPathArrow} />,
    table: (
      <TableToolbar
        profileType={profileSource?.ProfileType()}
        total={total}
        filtered={filtered}
        clearSelection={clearSelection}
        currentSearchString={currentSearchString}
      />
    ),
  };

  const hasProfileSource = profileSource !== undefined && profileSource.toString(timezone) !== '';

  return (
    <KeyDownProvider>
      <ProfileViewContextProvider value={{profileSource, compareMode}}>
        <DashboardProvider>
          <ProfileHeader
            profileSourceString={profileSource?.toString(timezone)}
            hasProfileSource={hasProfileSource}
            externalMainActions={profileViewExternalMainActions}
          />
          <VisualisationToolbar
            groupBy={groupBy}
            toggleGroupBy={toggleGroupBy}
            hasProfileSource={hasProfileSource}
            pprofdownloading={pprofDownloading}
            profileSource={profileSource}
            queryClient={queryClient}
            onDownloadPProf={onDownloadPProf}
            curPath={curPathArrow}
            setNewCurPath={setCurPathArrow}
            profileType={profileSource?.ProfileType()}
            total={total}
            filtered={filtered}
            currentSearchString={currentSearchString}
            setSearchString={setSearchString}
            groupByLabels={flamegraphData.metadataLabels ?? []}
            preferencesModal={preferencesModal}
            profileViewExternalSubActions={profileViewExternalSubActions}
            clearSelection={clearSelection}
            setGroupByLabels={setGroupByLabels}
            showVisualizationSelector={showVisualizationSelector}
          />

          {isColorStackLegendEnabled && (
            <ColorStackLegend
              compareMode={compareMode}
              mappings={colorMappings}
              loading={flamegraphData.metadataLoading}
            />
          )}

          <div className="w-full" ref={ref}>
            <DashboardLayout
              getDashboardItemByType={getDashboardItemByType}
              actionButtons={actionButtons}
            />
          </div>
        </DashboardProvider>
      </ProfileViewContextProvider>
    </KeyDownProvider>
  );
};
