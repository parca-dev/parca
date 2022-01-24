import {Fragment, useRef, useState} from 'react';
import cx from 'classnames';
import {Popover, Transition} from '@headlessui/react';
import DateTimeRangePickerTrigger from './DateTimeRangePickerTrigger';
import {DateTimeRange, DateUnion, POSITIONS, POSITION_TYPE} from './utils';
import {useClickAway} from 'react-use';
import DateTimeRangePickerPanel from './DateTimeRangePickerPanel';

import './style.css';

const getElementPosition = (element: HTMLElement | null) => {
  if (!element) {
    return null;
  }
  return element.offsetLeft + element.offsetWidth / 2;
};

const POPOVER_WIDTH = 384;

const DateTimeRangePicker = () => {
  const [range, setRange] = useState<DateTimeRange>(new DateTimeRange());
  const [isActive, setIsActive] = useState<boolean>(false);
  const [activePosition, setActivePosition] = useState<POSITION_TYPE>(POSITIONS.FROM);
  const containerRef = useRef<HTMLDivElement>(null);
  const fromRef = useRef<HTMLDivElement>(null);
  const toRef = useRef<HTMLDivElement>(null);
  useClickAway(containerRef, () => {
    setIsActive(false);
  });

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
              'absolute z-10 w-screen max-w-sm mt-2 rounded-lg shadow-lg ring-1 ring-black ring-opacity-5 arrow-top text-gray-100 dark:text-gray-800',
              {'left-12': activePosition === POSITIONS.TO}
            )}
            style={
              leftPosition
                ? {
                    left: leftPosition - POPOVER_WIDTH / 2,
                  }
                : undefined
            }
            static
          >
            <DateTimeRangePickerPanel
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
          </Popover.Panel>
        ) : null}
      </div>
    </Popover>
  );
};

export default DateTimeRangePicker;
