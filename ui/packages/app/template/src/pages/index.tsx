import * as React from 'react';
import {NextPage} from 'next';
import {Button} from 'components/Button';
import {capitalize} from '@parcaui/functions';

const Index: NextPage = () => {
  const handleClick = (): void => {
    alert('World');
  };

  return (
    <div>
      <Button label={capitalize('hello template')} onClick={handleClick} />
    </div>
  );
};

export default Index;
