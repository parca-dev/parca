import {Fragment, useState} from 'react';
import {Popover, Transition} from '@headlessui/react';
import {useAppSelector, selectDarkMode} from '@parca/store';
import {usePopper} from 'react-popper';

const transparencyValues = [-100, -80, -60, -40, -20, 0, 20, 40, 60, 80, 100];

const DiffLegendBar = ({
  onMouseEnter,
  onMouseLeave,
}: {
  onMouseEnter: () => void;
  onMouseLeave: () => void;
}) => {
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
    <div className="flex items-center m-2">
      {transparencyValues.map(value => {
        const valueAsPercentage = value / 100;
        const absoluteValue = Math.abs(valueAsPercentage);
        return (
          <div
            onMouseEnter={onMouseEnter}
            onMouseLeave={onMouseLeave}
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
  const [showLegendTooltip, setShowLegendTooltip] = useState(false);
  const [popperElement, setPopperElement] = useState<HTMLDivElement | null>(null);
  let [referenceElement, setReferenceElement] = useState<HTMLDivElement | null>(null);

  const {styles, attributes, ...popperProps} = usePopper(referenceElement, popperElement, {
    placement: 'auto-start',
    strategy: 'absolute',
  });

  const handleMouseEnter = () => {
    setShowLegendTooltip(true);
  };
  const handleMouseLeave = () => {
    setShowLegendTooltip(false);
  };

  return (
    <div className="mt-1 mb-2">
      <div ref={setReferenceElement} className="flex items-center justify-center">
        <span>Good</span>
        <DiffLegendBar onMouseEnter={handleMouseEnter} onMouseLeave={handleMouseLeave} />
        <span>Bad</span>
      </div>
      <Popover className="relative">
        {() => (
          <Transition
            show={showLegendTooltip}
            as={Fragment}
            enter="transition ease-out duration-200"
            enterFrom="opacity-0 translate-y-1"
            enterTo="opacity-100 translate-y-0"
            leave="transition ease-in duration-150"
            leaveFrom="opacity-100 translate-y-0"
            leaveTo="opacity-0 translate-y-1"
          >
            <Popover.Panel ref={setPopperElement} style={styles.popper} {...attributes.popper}>
              <div className="overflow-hidden rounded-lg shadow-lg ring-1 ring-black ring-opacity-5">
                <div className="p-4 bg-gray-50 dark:bg-gray-800">
                  <div className="flex items-center justify-center"></div>
                  <span className="block text-sm text-gray-500 dark:text-gray-50">
                    This is a differential icicle graph, where the purple means unchanged, and the
                    darker the red, the worse it got, and the darker the green, the better it got.
                  </span>
                </div>
              </div>
            </Popover.Panel>
          </Transition>
        )}
      </Popover>
    </div>
  );
};

export default DiffLegend;
