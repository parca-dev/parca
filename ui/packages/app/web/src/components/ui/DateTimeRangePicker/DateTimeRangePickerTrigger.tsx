import {formatDateStringForUI} from './utils';
import {Popover} from '@headlessui/react';

const Delimiter = () => <span className="mx-2">→</span>;

const DateTimeRangePickerTrigger = ({range, onClick, isActive}) => {
  return (
    <Popover.Button>
      <div
        onClick={onClick}
        className="relative flex justify-between w-[420px] bg-gray-50 dark:bg-gray-900 border-t border-r border-b border-l dark:border-gray-600 rounded-md shadow-sm pl-3 pr-10 py-2 text-left cursor-default focus:outline-none focus:ring-1 focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm"
      >
        {isActive ? (
          <div className="flex justify-center w-full">
            <button>{formatDateStringForUI(range.from)}</button>
            <Delimiter />
            <button>{formatDateStringForUI(range.to)}</button>
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
