import Select, {contructItemsFromArray} from '../../Select';
import {useState} from 'react';
import Input from '../../Input';
import Button from 'components/ui/Button';
import {RelativeDate, UNITS, UNIT_TYPE} from '../utils';
import {capitalizeFirstLetter} from 'libs/utils';

const constructKeyAndLabels = (UNITS: UNIT_TYPE[]) => {
  return UNITS.map(unit => ({
    key: unit,
    label: `${capitalizeFirstLetter(unit)} ago`,
  }));
};

type RelativeRangePickerProps = {
  date: RelativeDate;
  onChange?: (date: RelativeDate) => void;
};

const RelativeRangePicker = ({date, onChange = () => null}: RelativeRangePickerProps) => {
  const [unit, setUnit] = useState<UNIT_TYPE>(date.unit);
  const [value, setValue] = useState<number>(date.value);
  return (
    <div className="bg-gray-200 dark:bg-gray-800 ">
      <div className="flex justify-between p-1 py-8">
        <Input
          type="number"
          className="w-1/2"
          value={value}
          onChange={e => setValue(parseInt(e.target.value, 10))}
        />
        <Select
          width={40}
          items={contructItemsFromArray(constructKeyAndLabels(Object.values(UNITS)))}
          selectedKey={unit}
          onSelection={key => setUnit(key as UNIT_TYPE)}
        />
      </div>
      <div className="max-w-1/2 mx-auto py-2 pt-4">
        <Button
          onClick={() => {
            console.log(value, unit);
            onChange(new RelativeDate(unit, value));
          }}
        >
          Apply
        </Button>
      </div>
    </div>
  );
};

export default RelativeRangePicker;
