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
    <Popover.Panel className="">
      <div className="bg-gray-200 dark:bg-gray-800">
        <Tab
          tabs={['Absolute', 'Relative']}
          panels={[
            'Absolute',
            <RelativeRangePicker
              date={date as RelativeDate}
              onChange={date => onChange(date, position)}
            />,
          ]}
          defaultTabIndex={1}
        />
      </div>
    </Popover.Panel>
  );
};

export default DateTimeRangePickerPanel;
