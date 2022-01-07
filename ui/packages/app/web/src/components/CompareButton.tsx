import React, {useState} from 'react';
import {usePopper} from 'react-popper';
import Button from './ui/Button';

const CompareButton = ({disabled, onClick}: {disabled: boolean; onClick: () => void}) => {
  const [compareHover, setCompareHover] = useState<boolean>(false);
  const [comparePopperReferenceElement, setComparePopperReferenceElement] =
    useState<HTMLDivElement | null>(null);
  const [comparePopperElement, setComparePopperElement] = useState<HTMLDivElement | null>(null);
  const {styles, attributes} = usePopper(comparePopperReferenceElement, comparePopperElement, {
    placement: 'bottom',
  });

  const compareExplanation =
    'Compare two profiles and see the relative difference between them more clearly.';

  if (disabled) return <></>;

  return (
    <div ref={setComparePopperReferenceElement}>
      <Button
        color="neutral"
        disabled={disabled}
        onClick={onClick}
        onMouseEnter={() => setCompareHover(true)}
        onMouseLeave={() => setCompareHover(false)}
      >
        Compare
      </Button>
      {compareHover && (
        <div ref={setComparePopperElement} style={styles.popper} {...attributes.popper}>
          <div className="flex">
            <div className="relative mx-2">
              <svg className="text-black h-1 w-full left-0" x="0px" y="0px" viewBox="0 0 255 127.5">
                <polygon className="fill-current" points="0,127.5 127.5,0 255,127.5" />
              </svg>
              <div className="bg-black text-white text-xs rounded py-2 px-3 right-0 w-40">
                {compareExplanation}
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default CompareButton;
