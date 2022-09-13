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

import {useState, useEffect, Children} from 'react';
import {useContainerDimensions} from '@parca/dynamicsize';

interface Props {
  children: JSX.Element;
  [x: string]: any;
}

const addPropsToChildren = (children: JSX.Element, props: {[x: string]: any}): JSX.Element[] => {
  const addProps = (child: JSX.Element): JSX.Element => ({
    ...child,
    props: {
      ...child.props,
      ...props,
    },
  });

  return Children.map(children, addProps);
};

export const ResponsiveSvg = (props: Props): JSX.Element => {
  const {children} = props;
  const {ref, dimensions} = useContainerDimensions();
  const {width} = dimensions ?? {width: 0};
  const [height, setHeight] = useState(0);
  const childrenWithDimensions = addPropsToChildren(children, {width, height});

  useEffect(() => {
    if (ref.current != null) {
      setHeight(ref?.current.getBoundingClientRect().height);
    }
  }, [width, ref]);

  return (
    <div ref={ref} className="w-full">
      <svg viewBox={`0 0 ${width} ${height}`} preserveAspectRatio="xMidYMid meet" {...props}>
        {childrenWithDimensions}
      </svg>
    </div>
  );
};

export default ResponsiveSvg;
