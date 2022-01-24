import Select, {contructItemsFromArray} from 'components/ui/Select';
import {useState} from 'react';
import Input from 'components/ui/Input';
import DatePicker from 'react-datepicker';
import Button from 'components/ui/Button';
import {AbsoluteDate, getDateHoursAgo} from '../../utils';

import 'react-datepicker/dist/react-datepicker.css';

const CustomTimeInput = ({date, value, onChange}) => (
  <Input value={value} onChange={e => onChange(e.target.value)} />
);

type AbsoluteDatePickerProps = {
  date: AbsoluteDate;
  onChange?: (date: AbsoluteDate) => void;
};

const AbsoluteDatePicker = ({date, onChange = () => null}: AbsoluteDatePickerProps) => {
  const [value, setValue] = useState<Date>(date.value || getDateHoursAgo(1));
  return (
    <div className="bg-gray-200 dark:bg-gray-800 rounded p-2">
      <div className="flex justify-between p-1 py-8">
        <DatePicker selected={value} onChange={date => setValue(date)} showTimeInput inline />
      </div>
      <div className="max-w-1/2 mx-auto py-2 pt-4">
        <Button
          onClick={() => {
            console.log(value);
            onChange(new AbsoluteDate(value));
          }}
        >
          Apply
        </Button>
      </div>
    </div>
  );
};

export default AbsoluteDatePicker;
