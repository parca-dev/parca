import * as React from 'react';
import {render, screen} from '@testing-library/react';
import '@testing-library/jest-dom/extend-expect';
import {Button} from './Button';

test('loads and displays greeting', async () => {
  render(<Button label="Click Here" onClick={jest.fn()} />);

  expect(screen.getByTestId('button')).toHaveTextContent('Click Here');
});
