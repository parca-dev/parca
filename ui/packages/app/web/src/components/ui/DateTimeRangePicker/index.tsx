import {Fragment, useRef, useState} from 'react';
import {Popover, Transition} from '@headlessui/react';
import DateTimeRangePickerTrigger from './DateTimeRangePickerTrigger';
import {DateTimeRange, DateUnion, POSITIONS, POSITION_TYPE, RelativeDate} from './utils';
import {useClickAway} from 'react-use';
import DateTimeRangePickerPanel from './DateTimeRangePickerPanel';

const DateTimeRangePicker = () => {
  const [range, setRange] = useState<DateTimeRange>(new DateTimeRange());
  const [isActive, setIsActive] = useState<boolean>(false);
  const [activePosition, setActivePosition] = useState<POSITION_TYPE>(POSITIONS.FROM);
  const containerRef = useRef<HTMLDivElement>(null);
  useClickAway(containerRef, () => {
    setIsActive(false);
  });

  return (
    <Popover className="relative">
      <div ref={containerRef} className="relative items-center w-[330px] ">
        <DateTimeRangePickerTrigger
          range={range}
          isActive={isActive}
          onClick={() => {
            setIsActive(true);
          }}
        />
        <Transition
          as={Fragment}
          enter="transition ease-out duration-200"
          enterFrom="opacity-0 translate-y-1"
          enterTo="opacity-100 translate-y-0"
          leave="transition ease-in duration-150"
          leaveFrom="opacity-100 translate-y-0"
          leaveTo="opacity-0 translate-y-1"
        >
          <Popover.Panel className="absolute z-10 w-screen max-w-sm mt-2">
            <div className="overflow-hidden rounded-lg shadow-lg ring-1 ring-black ring-opacity-5">
              <DateTimeRangePickerPanel
                date={range.getDateForPosition(activePosition)}
                position={activePosition}
                onChange={(date: DateUnion, position: POSITION_TYPE) => {
                  range.setDateForPosition(date, position);
                  setRange(new DateTimeRange(range.from, range.to));
                }}
              />
            </div>
          </Popover.Panel>
        </Transition>
      </div>
    </Popover>
  );
};

export default DateTimeRangePicker;
