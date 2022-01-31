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
    <div className="flex flex-row w-[550px] bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-white">
      <div className=" border-r border-gray-200">
        <RelativeDatePicker
          key={position}
          position={position}
          range={range}
          onChange={(from, to) => onChange(from as DateUnion, to as DateUnion)}
        />
      </div>
      <AbsoluteDatePicker
        key={position}
        position={position}
        range={range}
        onChange={(from, to) => onChange(from as DateUnion, to as DateUnion)}
      />
    </div>
  );
};

export default DateTimeRangePickerPanelV2;
