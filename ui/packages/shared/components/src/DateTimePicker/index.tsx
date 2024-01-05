// Copyright 2022 The Parca Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import {useEffect, useState} from 'react';

import {Popover} from '@headlessui/react';
import {Icon} from '@iconify/react';
import cx from 'classnames';
import ReactDatePicker from 'react-datepicker';
import {usePopper} from 'react-popper';

import {convertLocalToUTCDate, convertUTCToLocalDate} from '@parca/utilities';

import {AbsoluteDate} from '../DateTimeRangePicker/utils';
import Input from '../Input';

export const DATE_FORMAT = 'yyyy-MM-DD HH:mm:ss';

const NOW = 'now';

export type ABSOLUTE_TIME_ALIASES_TYPE = typeof NOW;

export const ABSOLUTE_TIME_ALIASES: Record<string, ABSOLUTE_TIME_ALIASES_TYPE> = {
  NOW,
};

export type AbsoluteDateValue = Date | ABSOLUTE_TIME_ALIASES_TYPE;

interface Props {
  selected: AbsoluteDate;
  onChange: (date: AbsoluteDate) => void;
}

export const DateTimePicker = ({selected, onChange}: Props): JSX.Element => {
  const [referenceElement, setReferenceElement] = useState<HTMLDivElement | null>();
  const [popperElement, setPopperElement] = useState<HTMLDivElement | null>();
  const {styles, attributes} = usePopper(referenceElement, popperElement, {
    placement: 'bottom-end',
    strategy: 'absolute',
  });
  const [textInput, setTextInput] = useState<string>(selected.getUIString());
  const [isTextInputDirty, setIsTextInputDirty] = useState<boolean>(false);

  useEffect(() => {
    setTextInput(selected.getUIString());
  }, [selected]);

  return (
    <Popover>
      {({open}) => (
        <div className="flex items-center text-sm w-fit" ref={setReferenceElement}>
          <Input
            value={textInput}
            onAction={() => {}}
            actionButton={
              <Popover.Button
                className={cx('w-full h-full flex items-center justify-center rounded-md', {
                  '!bg-gray-200 dark:!bg-gray-700': open,
                  '!bg-gray-100 dark:!bg-gray-800': !open,
                })}
              >
                <Icon icon="mdi:calendar-month-outline" fontSize={20} />
              </Popover.Button>
            }
            onBlur={() => {
              if (!isTextInputDirty) {
                return;
              }
              setIsTextInputDirty(false);
              if (textInput === NOW) {
                onChange(new AbsoluteDate(textInput));
                return;
              }
              const date = new Date(textInput);
              if (isNaN(date.getTime())) {
                setTextInput(selected.getUIString());
                return;
              }
              onChange(new AbsoluteDate(convertLocalToUTCDate(date)));
            }}
            onChange={e => {
              setTextInput(e.target.value);
              setIsTextInputDirty(true);
            }}
          />

          <Popover.Panel
            ref={setPopperElement}
            style={styles.popper}
            {...attributes.popper}
            className="z-10"
          >
            <ReactDatePicker
              selected={convertUTCToLocalDate(selected.getTime())}
              onChange={date => {
                if (date == null) {
                  return;
                }
                onChange(new AbsoluteDate(convertLocalToUTCDate(date)));
                setIsTextInputDirty(false);
              }}
              showTimeInput
              dateFormat={DATE_FORMAT}
              className="h-[38px] w-full rounded-md border border-gray-200 p-2 text-center text-sm dark:border-gray-600 dark:bg-gray-900"
              inline
            />
          </Popover.Panel>
        </div>
      )}
    </Popover>
  );
};
