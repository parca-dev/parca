import Tab from '../../Tab';
import type {DateTimeRange, DateUnion} from '../utils';
import RelativeDatePicker from './RelativeDatePicker';
import AbsoluteDatePicker from './AbsoluteDatePicker';

interface DateTimeRangePickerProps {
  range: DateTimeRange;
  onChange?: (from: DateUnion, to: DateUnion) => void;
}

const DateTimeRangePickerPanel = ({range, onChange = () => null}: DateTimeRangePickerProps) => {
  return (
    <div className="w-[300px] p-2 bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-white">
      <Tab
        tabs={['Relative', 'Absolute']}
        panels={[
          <RelativeDatePicker
            range={range}
            onChange={(from, to) => onChange(from as DateUnion, to as DateUnion)}
          />,
          <AbsoluteDatePicker
            range={range}
            onChange={(from, to) => onChange(from as DateUnion, to as DateUnion)}
          />,
        ]}
        defaultTabIndex={range.from.isRelative() ? 0 : 1}
      />
    </div>
  );
};

export default DateTimeRangePickerPanel;
