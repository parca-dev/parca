import {useState} from 'react';
import DatePicker from 'react-datepicker';
import {AbsoluteDate, getDateHoursAgo, POSITIONS, POSITION_TYPE} from '../../utils';
import ApplyButton from '../ApplyButton';

import 'react-datepicker/dist/react-datepicker.css';

interface AbsoluteDatePickerProps {
  date: AbsoluteDate;
  onChange?: (date: AbsoluteDate) => void;
  position?: POSITION_TYPE;
}

const AbsoluteDatePicker = ({date, onChange = () => null, position}: AbsoluteDatePickerProps) => {
  const [value, setValue] = useState<Date>(
    date.isRelative() ? getDateHoursAgo(position === POSITIONS.FROM ? 1 : 0) : date.value
  );
  return (
    <div className="p-1">
      <div className="flex justify-center p-1">
        <DatePicker selected={value} onChange={date => setValue(date)} showTimeInput inline />
      </div>
      <div className="max-w-1/2 mx-auto p-2">
        <ApplyButton
          position={position}
          onClick={() => {
            onChange(new AbsoluteDate(value));
          }}
        >
          Apply
        </ApplyButton>
      </div>
    </div>
  );
};

export default AbsoluteDatePicker;
