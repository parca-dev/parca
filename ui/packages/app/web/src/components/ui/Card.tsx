import React from 'react';
import cx from 'classnames';

interface CardProps {
  children: React.ReactNode;
  className?: string;
}

const Card = ({children, className = ''}: CardProps) => {
  return (
    <div className="mx-auto">
      <div
        className={cx(
          'g-gray dark:bg-gray-700 shadow overflow-hidden sm:rounded-lg flex-1 flex-column',
          {[className]: className.length}
        )}
      >
        {children}
      </div>
    </div>
  );
};

const Header = ({children}) => {
  return (
    <div
      className="bg-gray-200 dark:bg-gray-800 px-4 py-4 sm:px-6"
      style={{justifyContent: 'space-between', alignItems: 'stretch'}}
    >
      {children}
    </div>
  );
};

const Body = ({children}) => {
  return <div className="p-4">{children}</div>;
};

export default Object.assign(Card, {
  Header,
  Body,
});
