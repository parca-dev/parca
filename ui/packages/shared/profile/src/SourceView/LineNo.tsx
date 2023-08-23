import {useEffect, useRef} from 'react';

interface Props {
  value: number;
  isCurrent?: boolean;
}

export const LineNo = ({value, isCurrent = false}: Props): JSX.Element => {
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (isCurrent) {
      ref.current?.scrollIntoView({behavior: 'smooth', block: 'center'});
    }
  }, [isCurrent]);

  return <code ref={ref}>{value.toString() + '\n'}</code>;
};
