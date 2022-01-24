import {Popover} from '@headlessui/react';
import Button from 'components/ui/Button';
import {POSITIONS} from '../utils';

const ApplyButton = ({position, onClick, children}) => {
  if (position === POSITIONS.FROM) {
    return <Button onClick={onClick}>{children}</Button>;
  }
  return (
    <span onClick={onClick}>
      <Popover.Button as={Button}>{children}</Popover.Button>
    </span>
  );
};

export default ApplyButton;
