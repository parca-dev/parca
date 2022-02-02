import cx from 'classnames';
import {DateTimeRange, formatDateStringForUI} from './utils';
import {Popover} from '@headlessui/react';
import ConditionalWrapper from 'components/ConditionalWrapper';

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
    <ConditionalWrapper
      condition={!isActive}
      wrapper={({children}) => <Popover.Button>{children}</Popover.Button>}
    >
      <div
        onClick={onClick}
        className={cx(
          'text-gray-600 dark:text-gray-300 relative flex justify-between min-w-[200px] bg-gray-50 dark:bg-gray-900 border-t border-r border-b border-l dark:border-gray-600 rounded-md shadow-sm px-3 py-2 text-left cursor-default focus:outline-none focus:ring-1 focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm',
          {'!justify-center': isActive}
        )}
      >
        {isActive && range.from.isRelative()
          ? `${formatDateStringForUI(range.from)} → ${formatDateStringForUI(range.to)}`
          : range.getRangeStringForUI()}
        {!isActive ? <span className="px-2 cursor-pointer">▼</span> : null}
      </div>
    </ConditionalWrapper>
  );
};

export default DateTimeRangePickerTrigger;
