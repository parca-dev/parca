import {useState} from 'react';
import DatePicker from 'react-datepicker';
import {
  AbsoluteDate,
  DateTimeRange,
  getDateHoursAgo,
  POSITIONS,
  POSITION_TYPE,
} from '../../../utils';
import ApplyButton from '../ApplyButton';

import 'react-datepicker/dist/react-datepicker.css';

interface AbsoluteDatePickerPropsV2 {
  range: DateTimeRange;
  onChange?: (from: AbsoluteDate, to: AbsoluteDate) => void;
  position?: POSITION_TYPE;
}

const AbsoluteDatePickerV2 = ({
  range,
  onChange = () => null,
  position,
}: AbsoluteDatePickerPropsV2) => {
  const [from, setFrom] = useState<Date>(
    range.from.isRelative()
      ? getDateHoursAgo(position === POSITIONS.FROM ? 1 : 0)
      : (range.from as AbsoluteDate).value
  );
  const [to, setTo] = useState<Date>(
    range.to.isRelative()
      ? getDateHoursAgo(position === POSITIONS.TO ? 1 : 0)
      : (range.to as AbsoluteDate).value
  );
  return (
    <div className="p-4">
      <div className="mb-2 hidden">
        <span className="uppercase text-xs text-gray-500">Absolute Range</span>
      </div>
      <div className="flex flex-col justify-center">
        <div className="mb-2">
          <div className="mb-2">
            <span className="uppercase text-xs text-gray-500">From:</span>
          </div>
          <DatePicker
            selected={from}
            onChange={date => setFrom(date)}
            showTimeInput
            dateFormat="MMMM d, yyyy h:mm aa"
            className="text-sm w-48 p-2 rounded-md border border-gray-200"
          />
        </div>
        <div className="mb-1">
          <div className="mb-2">
            <span className="uppercase text-xs text-gray-500">To:</span>
          </div>
          <DatePicker
            selected={to}
            onChange={date => setTo(date)}
            showTimeInput
            dateFormat="MMMM d, yyyy h:mm aa"
            className="text-sm w-48 p-2 rounded-md border border-gray-200"
          />
        </div>
      </div>
      <div className="w-32 mx-auto mt-4">
        <ApplyButton
          position={position}
          onClick={() => {
            onChange(new AbsoluteDate(from), new AbsoluteDate(to));
          }}
        >
          Apply
        </ApplyButton>
      </div>
    </div>
  );
};

export default AbsoluteDatePickerV2;
