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

import React, {useMemo, useRef, useState} from 'react';

import {AnimatePresence, motion} from 'framer-motion';

import {useURLState} from '@parca/components';

import {ProfileSource} from '../ProfileSource';
import {useDashboard} from '../ProfileView/context/DashboardContext';
import {useVisualizationState} from '../ProfileView/hooks/useVisualizationState';
import {SandwichData} from '../ProfileView/types/visualization';
import {CalleesSection} from './components/CalleesSection';
import {CallersSection} from './components/CallersSection';

interface Props {
  profileSource: ProfileSource;
  sandwichData: SandwichData;
}

const Sandwich = React.memo(function Sandwich({
  sandwichData,
  profileSource,
}: Props): React.JSX.Element {
  const {dashboardItems} = useDashboard();
  const [sandwichFunctionName] = useURLState<string | undefined>('sandwich_function_name');

  const callersRef = React.useRef<HTMLDivElement | null>(null);
  const calleesRef = React.useRef<HTMLDivElement | null>(null);
  const [isExpanded, setIsExpanded] = useState(false);
  const defaultMaxFrames = 10;

  const callersCalleesContainerRef = useRef<HTMLDivElement | null>(null);

  const {curPathArrow, setCurPathArrow} = useVisualizationState();

  const callersFlamegraphData = useMemo(() => sandwichData.callers, [sandwichData.callers]);
  const calleesFlamegraphData = useMemo(() => sandwichData.callees, [sandwichData.callees]);

  return (
    <section className="flex flex-row h-full w-full">
      <AnimatePresence>
        <motion.div
          className="h-full w-full"
          key="sandwich-loaded"
          initial={{display: 'none', opacity: 0}}
          animate={{display: 'block', opacity: 1}}
          transition={{duration: 0.5}}
        >
          <div className="relative flex flex-row">
            {sandwichFunctionName !== undefined ? (
              <div className="w-full flex flex-col" ref={callersCalleesContainerRef}>
                <CallersSection
                  callersRef={callersRef}
                  callersFlamegraphData={callersFlamegraphData}
                  profileSource={profileSource}
                  curPathArrow={curPathArrow}
                  setCurPathArrow={setCurPathArrow}
                  isExpanded={isExpanded}
                  setIsExpanded={setIsExpanded}
                  defaultMaxFrames={defaultMaxFrames}
                />
                <div className="h-4" />
                <CalleesSection
                  calleesRef={calleesRef}
                  calleesFlamegraphData={calleesFlamegraphData}
                  profileSource={profileSource}
                  curPathArrow={curPathArrow}
                  setCurPathArrow={setCurPathArrow}
                />
              </div>
            ) : (
              <div className="items-center justify-center flex h-full w-full">
                <p className="text-sm">
                  {dashboardItems.includes('table') ? (
                    'Please select a function to view its callers and callees.'
                  ) : (
                    <>
                      Use the right-click menu on the Flame{' '}
                      {dashboardItems.includes('flamegraph') ? 'Graph' : 'Chart'} to choose a
                      function to view its callers and callees.
                    </>
                  )}
                </p>
              </div>
            )}
          </div>
        </motion.div>
      </AnimatePresence>
    </section>
  );
});

export default Sandwich;
