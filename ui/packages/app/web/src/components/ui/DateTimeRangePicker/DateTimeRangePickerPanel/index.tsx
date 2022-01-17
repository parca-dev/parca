import {Popover} from '@headlessui/react';
import Tab from '../../Tab';
import RelativeRangePicker from './RelativeRangePicker';

const DateTimeRangePickerPanel = () => {
  return (
    <Popover.Panel className="">
      <div className="">
        {' '}
        <Tab
          tabs={['Absolute', 'Relative']}
          panels={['Absolute', <RelativeRangePicker />]}
          defaultTabIndex={1}
        />
      </div>
    </Popover.Panel>
  );
};

export default DateTimeRangePickerPanel;
