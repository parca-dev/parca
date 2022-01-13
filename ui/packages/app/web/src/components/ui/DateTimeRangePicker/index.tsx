import {useRef, useState} from 'react';
import DateTimeRangePickerTrigger from './DateTimeRangePickerTrigger';
import {DateTimeRange} from './utils';
import {useClickAway} from 'react-use';

const DateTimeRangePicker = () => {
  const [range, setRange] = useState<DateTimeRange>(new DateTimeRange());
  const [isActive, setIsActive] = useState<boolean>(false);
  const containerRef = useRef<HTMLDivElement>(null);
  useClickAway(containerRef, () => {
    setIsActive(false);
  });

  return (
    <div ref={containerRef} className="flex items-center">
      <DateTimeRangePickerTrigger
        range={range}
        isActive={isActive}
        onClick={() => {
          setIsActive(true);
        }}
      />
    </div>
  );
};

export default DateTimeRangePicker;
