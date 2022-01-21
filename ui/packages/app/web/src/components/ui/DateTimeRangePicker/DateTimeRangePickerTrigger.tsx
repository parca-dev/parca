import cx from 'classnames';
import {DateTimeRange, formatDateStringForUI, POSITIONS, POSITION_TYPE} from './utils';
import {Popover} from '@headlessui/react';

const Delimiter = () => <span className="mx-2">→</span>;

const PositionButton = ({isActive, position, onClick, date}) => {
  console.log('isActive', isActive);
  return (
    <button onClick={e => onClick(e, position)}>
      <span className={cx({underline: isActive})}>{formatDateStringForUI(date)}</span>
    </button>
  );
};

type DateTimeRangePickerTriggerProps = {
  range: DateTimeRange;
  onClick: (position: POSITION_TYPE) => void;
  activePosition: POSITION_TYPE;
  isActive: boolean;
};

const DateTimeRangePickerTrigger = ({
  range,
  onClick,
  isActive,
  activePosition,
}: DateTimeRangePickerTriggerProps) => {
  const buttonClick = (e, position: POSITION_TYPE) => {
    e.stopPropagation();
    onClick(position);
  };

  return (
    <Popover.Button>
      <div
        onClick={() => onClick(POSITIONS.FROM)}
        className="relative flex justify-between w-[420px] bg-gray-50 dark:bg-gray-900 border-t border-r border-b border-l dark:border-gray-600 rounded-md shadow-sm pl-3 pr-10 py-2 text-left cursor-default focus:outline-none focus:ring-1 focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm"
      >
        {isActive ? (
          <div className="flex justify-center w-full">
            <PositionButton
              isActive={activePosition === POSITIONS.FROM}
              position={POSITIONS.FROM}
              date={range.from}
              onClick={buttonClick}
            />
            <Delimiter />
            <PositionButton
              isActive={activePosition === POSITIONS.TO}
              position={POSITIONS.TO}
              date={range.to}
              onClick={buttonClick}
            />
          </div>
        ) : (
          <button>{range.getRangeStringForUI()}</button>
        )}
        {!isActive ? <span>Show dates</span> : null}
      </div>
    </Popover.Button>
  );
};

export default DateTimeRangePickerTrigger;
