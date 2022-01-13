import {formatDateStringForUI} from './utils';

const DateTimeRangePickerTrigger = ({range, onClick, isActive}) => {
  return (
    <div
      onClick={onClick}
      className="relative bg-gray-50 dark:bg-gray-900 border-t border-r border-b border-l dark:border-gray-600 rounded-md shadow-sm pl-3 pr-10 py-2 text-left cursor-default focus:outline-none focus:ring-1 focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm"
    >
      {isActive ? (
        <>
          <button>{formatDateStringForUI(range.from)}</button> →{' '}
          <button>{formatDateStringForUI(range.to)}</button>
        </>
      ) : (
        <button>{range.getRangeStringForUI()}</button>
      )}
    </div>
  );
};

export default DateTimeRangePickerTrigger;
