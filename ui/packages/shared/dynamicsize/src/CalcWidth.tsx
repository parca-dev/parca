import React from 'react';

class DebounceFunction {
  time: number;
  timeout: number | null;

  constructor(time: number) {
    this.time = time;
    this.timeout = null;
  }

  update = (func): null => {
    if (this.timeout != null) {
      clearTimeout(this.timeout);
    }

    this.timeout = window.setTimeout(() => {
      if (this.timeout != null) {
        clearTimeout(this.timeout);
        func();
      }
    }, this.time);

    return null;
  };
}

const registerResizeListener = (delay: number, func: () => void): (() => void) => {
  const viewportDebounce = new DebounceFunction(delay);
  const updateViewport = (): null => viewportDebounce.update(func);

  window.addEventListener('resize', updateViewport);

  return () => {
    return window.removeEventListener('resize', updateViewport);
  };
};

const addPropsToChildren = (children, props): any => {
  const addProps = (child): any => ({
    ...child,
    props: {
      ...child.props,
      ...props,
    },
  });

  return React.Children.map(children, addProps);
};

interface CalcWidthProps {
  throttle: number;
  delay: number;
  resizeEvent?: (width: number) => void;
  children: React.ReactNode;
}

interface CalcWidthState {
  width: number | null;
}

export class CalcWidth extends React.Component<CalcWidthProps, CalcWidthState> {
  static: {
    remove: () => void;
    wrapper: HTMLElement | null;
  };

  constructor(props) {
    super(props);

    this.state = {
      width: null,
    };

    this.static = {
      remove: () => {},
      wrapper: null,
    };
  }

  componentDidMount = (): void => {
    const {wrapper} = this.static;
    const {throttle} = this.props;
    const eventWrap = (): void => {
      if (wrapper != null) {
        this.checkWidth(wrapper);
      }
    };
    this.static.remove = registerResizeListener(throttle, eventWrap);
    this.checkWidth(wrapper);
  };

  componentWillUnmount = (): void => {
    const {remove} = this.static;
    return remove();
  };

  checkWidth = (node: HTMLElement | null): void => {
    const {resizeEvent} = this.props;
    const {width} = this.state;

    if (node != null && width !== node.offsetWidth) {
      this.setState({width: node.offsetWidth});
      if (resizeEvent != null) {
        resizeEvent(node.offsetWidth);
      }
    }
  };

  getWrapper = (node: HTMLElement): HTMLElement => {
    this.static.wrapper = node;
    return node;
  };

  render = (): JSX.Element => {
    const {width} = this.state;
    const {children: rawChildren} = this.props;
    const ref = (node: HTMLElement) => this.getWrapper(node);
    const children = addPropsToChildren(rawChildren, {width});

    if (width == null) {
      // @ts-expect-error
      return <div {...{ref}} />;
    }

    // @ts-expect-error
    return <div {...{ref}}>{children}</div>;
  };
}
