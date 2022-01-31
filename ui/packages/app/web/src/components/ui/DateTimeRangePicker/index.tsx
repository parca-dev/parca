import {useRef, useState} from 'react';
import cx from 'classnames';
import {Popover} from '@headlessui/react';
import DateTimeRangePickerTriggerv1 from './DateTimeRangePickerTrigger';
import {DateTimeRange, DateUnion, POSITIONS, POSITION_TYPE} from './utils';
import {useClickAway} from 'react-use';
import DateTimeRangePickerPanelv1 from './DateTimeRangePickerPanel';
import DateTimeRangePickerPanelv2 from './v2/DateTimeRangePickerPanel';
import DateTimeRangePickerTriggerv2 from './v2/DateTimeRangePickerTrigger';

import './style.css';

const getElementPosition = (element: HTMLElement | null) => {
  if (element == null) {
    return null;
  }
  return element.offsetLeft + element.offsetWidth / 2;
};

const POPOVER_WIDTH = 384;

const DateTimeRangePicker = ({isV2 = false}) => {
  const [range, setRange] = useState<DateTimeRange>(new DateTimeRange());
  const [isActive, setIsActive] = useState<boolean>(false);
  const [activePosition, setActivePosition] = useState<POSITION_TYPE>(POSITIONS.FROM);
  const containerRef = useRef<HTMLDivElement>(null);
  const fromRef = useRef<HTMLDivElement>(null);
  const toRef = useRef<HTMLDivElement>(null);
  useClickAway(containerRef, () => {
    setIsActive(false);
  });
  const DateTimeRangePickerTrigger = isV2
    ? DateTimeRangePickerTriggerv2
    : DateTimeRangePickerTriggerv1;

  const fromLeftPosition = getElementPosition(fromRef.current);
  const toLeftPosition = getElementPosition(toRef.current);

  const leftPosition = activePosition === POSITIONS.FROM ? fromLeftPosition : toLeftPosition;

  return (
    <Popover className="relative">
      <div ref={containerRef} className="relative items-center w-[330px] ">
        <DateTimeRangePickerTrigger
          range={range}
          isActive={isActive}
          activePosition={activePosition}
          onClick={position => {
            setIsActive(true);
            setActivePosition(position);
          }}
          fromRef={fromRef}
          toRef={toRef}
        />
        {isActive ? (
          <Popover.Panel
            className={cx(
              'absolute z-10 w-fit mt-2 rounded shadow-lg ring-1 ring-black ring-opacity-5 arrow-top text-gray-100 dark:text-gray-800',
              {'left-12': activePosition === POSITIONS.TO}
            )}
            style={
              leftPosition != null && !Number.isNaN(leftPosition)
                ? {
                    left: leftPosition - POPOVER_WIDTH / 2,
                  }
                : undefined
            }
            static
          >
            {!isV2 ? (
              <DateTimeRangePickerPanelv1
                date={range.getDateForPosition(activePosition)}
                position={activePosition}
                onChange={(date: DateUnion, position: POSITION_TYPE) => {
                  range.setDateForPosition(date, position);
                  setRange(new DateTimeRange(range.from, range.to));
                  if (position === POSITIONS.FROM) {
                    setActivePosition(POSITIONS.TO);
                  } else {
                    setIsActive(false);
                  }
                }}
              />
            ) : (
              <DateTimeRangePickerPanelv2
                range={range}
                position={activePosition}
                onChange={(from: DateUnion, to: DateUnion) => {
                  setRange(new DateTimeRange(from, to));
                  setIsActive(false);
                }}
              />
            )}
          </Popover.Panel>
        ) : null}
      </div>
    </Popover>
  );
};

export default DateTimeRangePicker;
