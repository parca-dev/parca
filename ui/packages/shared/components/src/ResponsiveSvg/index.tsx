import {useEffect, useState, Children, ReactNode} from 'react';
import {useContainerDimensions} from '@parca/dynamicsize';

interface Props {
  children: ReactNode;
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

const ResponsiveSvg = (props: Props) => {
  const {children} = props;
  const {ref, dimensions} = useContainerDimensions();
  const {width} = dimensions ?? {width: 0};
  const [height, setHeight] = useState(0);
  const childrenWithDimensions = addPropsToChildren(children, {width, height});

  useEffect(() => {
    if (ref.current != null) {
      setHeight(ref?.current.getBoundingClientRect().height);
    }
  }, [width]);

  return (
    <div ref={ref} className="w-full">
      <svg viewBox={`0 0 ${width} ${height}`} preserveAspectRatio="xMidYMid meet" {...props}>
        {childrenWithDimensions}
      </svg>
    </div>
  );
};

export default ResponsiveSvg;
