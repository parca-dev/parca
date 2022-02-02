import {Popover} from '@headlessui/react';
import Button from 'components/ui/Button';

const ApplyButton = ({onClick, children}) => {
  return (
    <span onClick={onClick}>
      <Popover.Button as={Button}>{children}</Popover.Button>
    </span>
  );
};

export default ApplyButton;
