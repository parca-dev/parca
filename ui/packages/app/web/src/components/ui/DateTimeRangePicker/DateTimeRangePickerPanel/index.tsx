import {Popover} from '@headlessui/react';
import Tab from '../../Tab';
import type {AbsoluteDate, DateUnion, POSITION_TYPE, RelativeDate} from '../utils';
import RelativeDatePicker from './RelativeDatePicker';
import AbsoluteDatePicker from './AbsoluteDatePicker';

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
  console.log('DateTimeRangePickerPanel', date, position, date.isRelative());
  return (
    <Popover.Panel className="bg-gray-100 dark:bg-gray-800 p-1 text-black dark:text-white">
      <Tab
        tabs={['Absolute', 'Relative']}
        panels={[
          <AbsoluteDatePicker
            key={position}
            position={position}
            date={date as AbsoluteDate}
            onChange={date => onChange(date, position)}
          />,
          <RelativeDatePicker
            key={position}
            position={position}
            date={date as RelativeDate}
            onChange={date => onChange(date, position)}
          />,
        ]}
        defaultTabIndex={date.isRelative() ? 1 : 0}
        key={position}
      />
    </Popover.Panel>
  );
};

export default DateTimeRangePickerPanel;
