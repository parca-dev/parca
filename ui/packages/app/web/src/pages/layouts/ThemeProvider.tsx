import {useStore} from '../../store';
import {selectUi} from '../../store/ui.state';
import cx from 'classnames';
import React from 'react';

const ThemeProvider = ({children}: {children: React.ReactNode}) => {
  const {darkMode} = useStore(selectUi);

  return (
    <div className={cx(darkMode && 'dark')}>
      <div
        style={{minHeight: '100vh'}}
        className="bg-gray-50 text-gray-800 dark:bg-gray-900 dark:text-gray-200"
      >
        {children}
      </div>
    </div>
  );
};

export default ThemeProvider;
