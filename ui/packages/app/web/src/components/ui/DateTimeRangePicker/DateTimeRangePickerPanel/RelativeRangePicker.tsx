import Select, {contructItemsFromArray} from '../../Select';
import {useState} from 'react';
import Input from '../../Input';

const units = [
  {
    label: 'Minutes ago',
    key: 'minutes',
  },
  {
    label: 'Hours ago',
    key: 'hours',
  },
  {
    label: 'Days ago',
    key: 'days',
  },
];

const RelativeRangePicker = () => {
  const [unit, setUnit] = useState<string>(units[2].key);
  const [value, setValue] = useState<number>(1);
  return (
    <div className="bg-gray-200 dark:bg-gray-800">
      <div className="flex justify-between p-1">
        <Input
          type="number"
          className="w-1/2"
          value={value}
          onChange={e => setValue(parseInt(e.target.value, 10))}
        />
        <Select
          width={40}
          items={contructItemsFromArray(units)}
          selectedKey={unit}
          onSelection={key => setUnit(key || '')}
        />
      </div>
    </div>
  );
};

export default RelativeRangePicker;
