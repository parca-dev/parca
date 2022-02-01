import Tab from '../../../Tab';
import type {
  AbsoluteDate,
  DateTimeRange,
  DateUnion,
  POSITION_TYPE,
  RelativeDate,
} from '../../utils';
import RelativeDatePicker from './RelativeDatePicker';
import AbsoluteDatePicker from './AbsoluteDatePicker';

interface DateTimeRangePickerPropsV2 {
  range: DateTimeRange;
  position: POSITION_TYPE;
  onChange?: (from: DateUnion, to: DateUnion) => void;
}

const DateTimeRangePickerPanelV2 = ({
  range,
  position,
  onChange = () => null,
}: DateTimeRangePickerPropsV2) => {
  return (
    <div className="w-[300px] p-2 bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-white">
      <Tab
        tabs={['Absolute', 'Relative']}
        panels={[
          <AbsoluteDatePicker
            key={position}
            position={position}
            range={range}
            onChange={(from, to) => onChange(from as DateUnion, to as DateUnion)}
          />,
          <RelativeDatePicker
            key={position}
            position={position}
            range={range}
            onChange={(from, to) => onChange(from as DateUnion, to as DateUnion)}
          />,
        ]}
        defaultTabIndex={range.from.isRelative() ? 1 : 0}
        key={position}
      />
    </div>
  );
};

export default DateTimeRangePickerPanelV2;
