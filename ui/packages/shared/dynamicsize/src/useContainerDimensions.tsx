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

import {RefObject, useRef, useState, useEffect} from 'react';

export const useContainerDimensions = (): {
  dimensions?: DOMRect;
  ref: RefObject<HTMLDivElement>;
} => {
  const ref = useRef<HTMLDivElement>(null);
  const [dimensions, setDimensions] = useState<DOMRect>();

  const updateDimensions = (): void => setDimensions(ref.current?.getBoundingClientRect());

  useEffect(() => {
    updateDimensions();
    window.addEventListener('resize', updateDimensions);
    return () => window.removeEventListener('resize', updateDimensions);
  }, []);

  return {dimensions, ref};
};
