import React from 'react';

const Card = ({children}: {children: React.ReactNode}) => {
  return (
    <div className="mx-auto">
      <div className="bg-gray dark:bg-gray-700 shadow overflow-hidden sm:rounded-lg flex-1 flex-column">
        {children}
      </div>
    </div>
  );
};

const Header = ({children}: {children: React.ReactNode}) => {
  return (
    <div
      className="bg-gray-200 dark:bg-gray-800 px-4 py-4 sm:px-6"
      style={{justifyContent: 'space-between', alignItems: 'stretch'}}
    >
      {children}
    </div>
  );
};

const Body = ({children}: {children: React.ReactNode}) => {
  return <div className="p-4">{children}</div>;
};

export default Object.assign(Card, {
  Header,
  Body,
});
