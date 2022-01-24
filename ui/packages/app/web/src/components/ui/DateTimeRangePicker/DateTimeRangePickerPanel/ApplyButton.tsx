import {Popover} from '@headlessui/react';
import Button from 'components/ui/Button';
import {POSITIONS} from '../utils';

const ApplyButton = ({position, onClick}) => {
  if (position === POSITIONS.FROM) {
    return <Button onClick={onClick}>Apply</Button>;
  }
  return (
    <span onClick={onClick}>
      <Popover.Button as={Button}>Apply</Popover.Button>
    </span>
  );
};

export default ApplyButton;
