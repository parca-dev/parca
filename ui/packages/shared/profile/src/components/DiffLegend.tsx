import {Fragment} from 'react';
import {Popover, Transition} from '@headlessui/react';
import {XIcon} from '@heroicons/react/solid';
import {useAppSelector, selectDarkMode} from '@parca/store';

const transparencyValues = [-100, -80, -60, -40, -20, 0, 20, 40, 60, 80, 100];

const DiffLegendBar = () => {
  const isDarkMode = useAppSelector(selectDarkMode);

  const newSpanColor = isDarkMode ? '#B3BAE1' : '#929FEB';

  const getIncreasedSpanColor = (transparency: number) => {
    return isDarkMode
      ? `rgba(255, 177, 204, ${transparency})`
      : `rgba(254, 153, 187, ${transparency})`;
  };
  const getReducedSpanColor = (transparency: number) => {
    return isDarkMode
      ? `rgba(103, 158, 92, ${transparency})`
      : `rgba(164, 214, 153, ${transparency})`;
  };

  return (
    <div className="flex items-center mb-2 mt-2 ml-2 mr-2">
      {transparencyValues.map(value => {
        const valueAsPercentage = value / 100;
        const absoluteValue = Math.abs(valueAsPercentage);
        return (
          <div
            className="w-8 h-4"
            key={valueAsPercentage}
            style={{
              backgroundColor:
                absoluteValue === 0
                  ? newSpanColor
                  : valueAsPercentage > 0
                  ? getIncreasedSpanColor(absoluteValue)
                  : getReducedSpanColor(absoluteValue),
            }}
          ></div>
        );
      })}
    </div>
  );
};

const DiffLegend = () => {
  return (
    <div className="fixed bottom-2 right-7">
      <Popover className="relative">
        {({open}) => (
          <>
            <Popover.Button>
              <div className="rounded-md p-2 cursor-pointer bg-gray-200 dark:bg-gray-800 hover:underline">
                {open ? <XIcon width={20} /> : <span>Show Legend</span>}
              </div>
            </Popover.Button>
            <Popover.Overlay className="bg-black opacity-50 fixed inset-0" />
            <Transition
              as={Fragment}
              enter="transition ease-out duration-200"
              enterFrom="opacity-0 translate-y-1"
              enterTo="opacity-100 translate-y-0"
              leave="transition ease-in duration-150"
              leaveFrom="opacity-100 translate-y-0"
              leaveTo="opacity-0 translate-y-1"
            >
              <Popover.Panel
                as="menu"
                className="absolute z-10 -top-[8rem] right-0 w-screen max-w-sm lg:max-w-3xl"
              >
                <div className="overflow-hidden rounded-lg shadow-lg ring-1 ring-black ring-opacity-5">
                  <div className="p-4 bg-gray-50 dark:bg-gray-800">
                    <div className="flex items-center justify-center">
                      <span>Good</span>
                      <DiffLegendBar />
                      <span>Bad</span>
                    </div>
                    <span className="block text-sm text-gray-500 dark:text-gray-50">
                      This is a differential icicle graph, where the purple means unchanged, and the
                      darker the red, the worse it got, and the darker the green, the better it got.
                    </span>
                  </div>
                </div>
              </Popover.Panel>
            </Transition>
          </>
        )}
      </Popover>
    </div>
  );
};

export default DiffLegend;
