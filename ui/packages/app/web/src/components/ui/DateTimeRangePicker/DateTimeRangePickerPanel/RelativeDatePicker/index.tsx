import Select, {contructItemsFromArray} from 'components/ui/Select';
import {useState} from 'react';
import Input from 'components/ui/Input';
import Button from 'components/ui/Button';
import {POSITION_TYPE, RelativeDate, UNITS, UNIT_TYPE} from '../../utils';
import {capitalizeFirstLetter} from 'libs/utils';
import ConditionalWrapper from 'components/ConditionalWrapper';
import {Popover} from '@headlessui/react';
import ApplyButton from '../ApplyButton';

const constructKeyAndLabels = (UNITS: UNIT_TYPE[]) => {
  return UNITS.map(unit => ({
    key: unit,
    label: `${capitalizeFirstLetter(unit)} ago`,
  }));
};

type RelativeDatePickerProps = {
  date: RelativeDate;
  onChange?: (date: RelativeDate) => void;
  position?: POSITION_TYPE;
};

const RelativeDatePicker = ({date, onChange = () => null, position}: RelativeDatePickerProps) => {
  const [unit, setUnit] = useState<UNIT_TYPE>(date.isRelative() ? date.unit : UNITS.HOUR);
  const [value, setValue] = useState<number>(date.isRelative() ? date.value : 1);
  return (
    <div className="bg-gray-200 dark:bg-gray-800 rounded p-2">
      <div className="flex justify-between p-1 py-4">
        <Input
          type="number"
          className="w-1/2 mr-2"
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
      <div className="max-w-1/2 mx-auto pb-2">
        <ApplyButton
          position={position}
          onClick={() => {
            onChange(new RelativeDate(unit, value));
          }}
        >
          Apply
        </ApplyButton>
      </div>
      <div className="my-4 mt-8 border-b-2 border-gray-300 text-gray-400 text-xs leading-[0px] mx-auto text-center">
        <span className="bg-gray-200 dark:bg-gray-800 px-1">OR</span>
      </div>
      <div className="max-w-1/2 mx-auto py-2">
        <ApplyButton
          position={position}
          onClick={() => {
            onChange(new RelativeDate(UNITS.MINUTE, 0));
          }}
        >
          Set to 'NOW'
        </ApplyButton>
      </div>
      <p className="text-gray-500 text-xs italic text-center mx-14">
        Note: Setting to 'NOW' means that on every search the time will be set to the time of the
        search.
      </p>
    </div>
  );
};

export default RelativeDatePicker;
