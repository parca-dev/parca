import {Fragment, useRef, useState} from 'react';
import cx from 'classnames';
import {Tab as HeadlessTab} from '@headlessui/react';

const Tab = ({tabs, panels, defaultTabIndex = 0}) => {
  return (
    <HeadlessTab.Group defaultIndex={defaultTabIndex}>
      <HeadlessTab.List className="flex p-1 space-x-1 bg-blue-900/20 rounded-xl w-[80%] mx-auto">
        {tabs.map((tab, idx) => (
          <HeadlessTab
            key={idx}
            className={({selected}) =>
              cx(
                'w-full py-2.5 text-sm leading-5 font-medium text-blue-700 rounded-lg',
                'focus:outline-none focus:ring-2 ring-offset-2 ring-offset-blue-400 ring-white ring-opacity-60',
                selected
                  ? 'bg-white shadow'
                  : 'text-blue-100 hover:bg-white/[0.12] hover:text-white'
              )
            }
          >
            {tab}
          </HeadlessTab>
        ))}
      </HeadlessTab.List>
      <HeadlessTab.Panels className="mt-2">
        {panels.map((panel, idx) => (
          <HeadlessTab.Panel
            key={idx}
            className={cx(
              'bg-white rounded-xl p-3',
              'focus:outline-none focus:ring-2 ring-offset-2 ring-offset-blue-400 ring-white ring-opacity-60'
            )}
          >
            {panel}
          </HeadlessTab.Panel>
        ))}
      </HeadlessTab.Panels>
    </HeadlessTab.Group>
  );
};

export default Tab;
