import {Popover} from '@headlessui/react';
import Tab from '../../Tab';
import type {DateUnion, POSITION_TYPE, RelativeDate} from '../utils';
import RelativeRangePicker from './RelativeRangePicker';

type DateTimeRangePickerProps = {
  date: DateUnion;
  position: POSITION_TYPE;
  onChange?: (date: DateUnion, position: POSITION_TYPE) => void;
};

const DateTimeRangePickerPanel = ({
  date,
  position,
  onChange = () => null,
}: DateTimeRangePickerProps) => {
  return (
    <Popover.Panel className="bg-gray-100 dark:bg-gray-800 p-4 text-black dark:text-white">
      <Tab
        tabs={['Absolute', 'Relative']}
        panels={[
          'Absolute',
          <RelativeRangePicker
            key={position}
            date={date as RelativeDate}
            onChange={date => onChange(date, position)}
          />,
        ]}
        defaultTabIndex={date.isRelative() ? 1 : 0}
      />
    </Popover.Panel>
  );
};

export default DateTimeRangePickerPanel;
