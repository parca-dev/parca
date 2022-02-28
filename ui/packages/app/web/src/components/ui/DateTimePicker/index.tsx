import {convertLocalToUTCDate, convertUTCToLocalDate} from '@parca/functions';
import ReactDatePicker from 'react-datepicker';

const DateTimePicker = ({selected, onChange}) => (
  <ReactDatePicker
    selected={selected}
    onChange={onChange}
    showTimeInput
    dateFormat="MMMM d, yyyy h:mm aa"
    className="text-sm w-52 p-2 rounded-md  bg-gray-50 dark:bg-gray-900 border border-gray-200 dark:border-gray-600"
  />
);

export const UTCDateTimePicker = ({selected, onChange}) => (
  <ReactDatePicker
    selected={convertUTCToLocalDate(selected)}
    onChange={date => onChange(convertLocalToUTCDate(date))}
    showTimeInput
    dateFormat="MMMM d, yyyy h:mm aa"
    className="text-sm w-52 p-2 rounded-md  bg-gray-50 dark:bg-gray-900 border border-gray-200 dark:border-gray-600"
  />
);

export default DateTimePicker;
