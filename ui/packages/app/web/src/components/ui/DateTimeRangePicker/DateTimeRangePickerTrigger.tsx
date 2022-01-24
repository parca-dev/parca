import cx from 'classnames';
import {DateTimeRange, formatDateStringForUI, POSITIONS, POSITION_TYPE} from './utils';
import {Popover} from '@headlessui/react';
import ConditionalWrapper from 'components/ConditionalWrapper';

const Delimiter = () => <span className="mx-2">→</span>;

const PositionButton = ({isActive, position, onClick, date, buttonRef}) => {
  return (
    <button onClick={e => onClick(e, position)} ref={buttonRef}>
      <span className={cx({underline: isActive})}>{formatDateStringForUI(date)}</span>
    </button>
  );
};

type DateTimeRangePickerTriggerProps = {
  range: DateTimeRange;
  onClick: (position: POSITION_TYPE) => void;
  activePosition: POSITION_TYPE;
  isActive: boolean;
  fromRef: React.RefObject<HTMLDivElement>;
  toRef: React.RefObject<HTMLDivElement>;
};

const DateTimeRangePickerTrigger = ({
  range,
  onClick,
  isActive,
  activePosition,
  fromRef,
  toRef,
}: DateTimeRangePickerTriggerProps) => {
  const buttonClick = (e, position: POSITION_TYPE = POSITIONS.FROM) => {
    e.stopPropagation();
    e.preventDefault();
    onClick(position);
  };

  return (
    <ConditionalWrapper
      condition={!isActive}
      wrapper={({children}) => <Popover.Button>{children}</Popover.Button>}
    >
      <div
        onClick={() => {
          if (isActive) {
            //return;
          }
          onClick(POSITIONS.FROM);
        }}
        className="relative flex justify-between w-[400px] bg-gray-50 dark:bg-gray-900 border-t border-r border-b border-l dark:border-gray-600 rounded-md shadow-sm px-3 py-2 text-left cursor-default focus:outline-none focus:ring-1 focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm"
      >
        {isActive ? (
          <div className="flex justify-center w-full">
            <PositionButton
              isActive={activePosition === POSITIONS.FROM}
              position={POSITIONS.FROM}
              date={range.from}
              onClick={buttonClick}
              buttonRef={fromRef}
            />
            <Delimiter />
            <PositionButton
              isActive={activePosition === POSITIONS.TO}
              position={POSITIONS.TO}
              date={range.to}
              onClick={buttonClick}
              buttonRef={toRef}
            />
          </div>
        ) : (
          <button>{range.getRangeStringForUI()}</button>
        )}
        {!isActive ? <span className="px-2 cursor-pointer">▼</span> : null}
      </div>
    </ConditionalWrapper>
  );
};

export default DateTimeRangePickerTrigger;
