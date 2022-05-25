import cx from 'classnames';
import {DateTimeRange, formatDateStringForUI} from './utils';
import {Popover} from '@headlessui/react';

interface DateTimeRangePickerTriggerProps {
  range: DateTimeRange;
  onClick: () => void;
  isActive: boolean;
}

const DateTimeRangePickerTrigger = ({
  range,
  onClick,
  isActive,
}: DateTimeRangePickerTriggerProps) => {
  return (
    <>
      <Popover.Button onClick={onClick}>
        <div
          onClick={onClick}
          className={cx(
            'text-gray-600 dark:text-gray-300 relative flex justify-between min-w-[200px] border-t border-r border-b border-l dark:border-gray-600 rounded-md shadow-sm px-3 py-2 text-left cursor-default focus:outline-none focus:ring-1 focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm',
            {'bg-gray-50 dark:bg-gray-900': !isActive},
            {'!justify-center, bg-gray-100 dark:bg-gray-800': isActive}
          )}
        >
          <span className="w-[147px] text-ellipsis overflow-hidden whitespace-nowrap">
            {isActive && range.from.isRelative()
              ? `${formatDateStringForUI(range.from)} → ${formatDateStringForUI(range.to)}`
              : range.getRangeStringForUI()}
          </span>

          <span className="px-2 cursor-pointer">{!isActive ? '▼' : '▲'}</span>
        </div>
      </Popover.Button>
    </>
  );
};

export default DateTimeRangePickerTrigger;
