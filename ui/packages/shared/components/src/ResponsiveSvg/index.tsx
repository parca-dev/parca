import {Children, ReactChild} from 'react';
import {useContainerDimensions} from '@parca/dynamicsize';

interface Props {
  children: ReactChild;
  [x: string]: any;
}

const addPropsToChildren = (children, props): any => {
  const addProps = (child): any => ({
    ...child,
    props: {
      ...child.props,
      ...props,
    },
  });

  return Children.map(children, addProps);
};

export const ResponsiveSvg = (props: Props) => {
  const {children} = props;
  const {ref, dimensions} = useContainerDimensions();
  const {width, height} = dimensions ?? {width: 0, height: 0};
  const childrenWithDimensions = addPropsToChildren(children, {width, height});

  return (
    <div ref={ref} className="w-full">
      <svg viewBox={`0 0 ${width} ${height}`} preserveAspectRatio="xMidYMid meet" {...props}>
        {childrenWithDimensions}
      </svg>
    </div>
  );
};
