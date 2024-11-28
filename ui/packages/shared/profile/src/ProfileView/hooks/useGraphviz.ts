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

import {useEffect, useState} from 'react';

import graphviz from 'graphviz-wasm';

import {Callgraph as CallgraphType} from '@parca/client';

import {jsonToDot} from '../../Callgraph/utils';

interface UseGraphvizProps {
  callgraphData?: CallgraphType;
  width?: number;
  colorRange: [string, string];
}

export const useGraphviz = ({
  callgraphData,
  width,
  colorRange,
}: UseGraphvizProps): {
  graphvizLoaded: boolean;
  callgraphSVG: string | undefined;
} => {
  const [graphvizLoaded, setGraphvizLoaded] = useState(false);
  const [callgraphSVG, setCallgraphSVG] = useState<string | undefined>(undefined);

  useEffect(() => {
    async function loadGraphviz(): Promise<void> {
      await graphviz.loadWASM();
      setGraphvizLoaded(true);
    }
    void loadGraphviz();
  }, []);

  useEffect(() => {
    async function loadCallgraphSVG(
      graph: CallgraphType,
      width: number,
      colorRange: [string, string]
    ): Promise<void> {
      await setCallgraphSVG(undefined);
      const dataAsDot = await jsonToDot({
        graph,
        width,
        colorRange,
      });
      const svgGraph = await graphviz.layout(dataAsDot, 'svg', 'dot');
      await setCallgraphSVG(svgGraph);
    }

    if (graphvizLoaded && callgraphData != null && width != null) {
      void loadCallgraphSVG(callgraphData, width, colorRange);
    }
  }, [graphvizLoaded, callgraphData, width, colorRange]);

  return {graphvizLoaded, callgraphSVG};
};
