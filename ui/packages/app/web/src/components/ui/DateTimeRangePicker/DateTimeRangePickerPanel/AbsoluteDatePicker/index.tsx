import Select, {contructItemsFromArray} from 'components/ui/Select';
import {useState} from 'react';
import Input from 'components/ui/Input';
import DatePicker from 'react-datepicker';
import Button from 'components/ui/Button';
import {AbsoluteDate, getDateHoursAgo, POSITIONS, POSITION_TYPE} from '../../utils';

import 'react-datepicker/dist/react-datepicker.css';
import ApplyButton from '../ApplyButton';

const CustomTimeInput = ({date, value, onChange}) => (
  <Input value={value} onChange={e => onChange(e.target.value)} />
);

type AbsoluteDatePickerProps = {
  date: AbsoluteDate;
  onChange?: (date: AbsoluteDate) => void;
  position?: POSITION_TYPE;
};

const AbsoluteDatePicker = ({date, onChange = () => null, position}: AbsoluteDatePickerProps) => {
  const [value, setValue] = useState<Date>(
    date.isRelative() ? getDateHoursAgo(position === POSITIONS.FROM ? 1 : 0) : date.value
  );
  return (
    <div className="bg-gray-200 dark:bg-gray-800 rounded p-1">
      <div className="flex justify-center p-1">
        <DatePicker selected={value} onChange={date => setValue(date)} showTimeInput inline />
      </div>
      <div className="max-w-1/2 mx-auto p-2">
        <ApplyButton
          position={position}
          onClick={() => {
            console.log(value);
            onChange(new AbsoluteDate(value));
          }}
        />
      </div>
    </div>
  );
};

export default AbsoluteDatePicker;
