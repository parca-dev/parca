import {useState} from 'react';
import {DateTimeRange} from '../utils';
import DateTimeRangePicker from '../index';

const StateWrappedComponent = props => {
  const [range, setRange] = useState(new DateTimeRange());
  return <DateTimeRangePicker range={range} onRangeSelection={setRange} {...props} />;
};

export default StateWrappedComponent;
