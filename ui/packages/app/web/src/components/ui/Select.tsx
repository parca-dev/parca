import { Listbox, Transition } from '@headlessui/react'
import { CheckIcon, SelectorIcon } from '@heroicons/react/solid'
import cx from 'classnames'
import { Fragment } from 'react'

export interface SelectItem {
  key: string
  label: string
}

const Select = ({
  items,
  selectedKey,
  onSelection,
  placeholder,
  width
}: {
  items: SelectItem[]
  selectedKey: string | undefined
  onSelection: (value: string | undefined) => void
  placeholder?: string
  width?: number
}) => {
  const selection = items.find(v => v.key === selectedKey)

  return (
    <>
      <Listbox value={selectedKey} onChange={onSelection}>
        {({ open }) => (
          <>
            <div>
              <Listbox.Button
                className={cx(
                  width !== undefined ? `w-${width}` : '',
                  'relative bg-white border border-gray-300 rounded-md shadow-sm pl-3 pr-10 py-2 text-left cursor-default focus:outline-none focus:ring-1 focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm'
                )}
              >
                <span className='flex items-center'>
                  {/* SLOT */}
                  <span className='ml-3 block truncate'>{selection?.label || placeholder}</span>
                </span>
                <span className='ml-3 absolute inset-y-0 right-0 flex items-center pr-2 pointer-events-none'>
                  <SelectorIcon className='h-5 w-5 text-gray-400' aria-hidden='true' />
                </span>
              </Listbox.Button>

              <Transition
                show={open}
                as={Fragment}
                leave='transition ease-in duration-100'
                leaveFrom='opacity-100'
                leaveTo='opacity-0'
              >
                <Listbox.Options
                  className={cx(
                    width !== undefined ? `w-${width}` : '',
                    'absolute z-10 mt-1 bg-white shadow-lg rounded-md py-1 text-base ring-1 ring-black ring-opacity-5 overflow-auto focus:outline-none sm:text-sm'
                  )}
                >
                  {items.map(option => (
                    <Listbox.Option
                      key={option.key}
                      className={({ active }) =>
                        cx(
                          active ? 'text-white bg-indigo-600' : 'text-gray-900',
                          'cursor-default select-none relative py-2 pl-3 pr-9'
                        )
                      }
                      value={option.key}
                    >
                      {({ selected, active }) => (
                        <>
                          <div className='flex items-center'>
                            <span
                              className={cx(
                                selected ? 'font-semibold' : 'font-normal',
                                'ml-3 block truncate'
                              )}
                            >
                              {option.label}
                            </span>
                          </div>

                          {selected ? (
                            <span
                              className={cx(
                                active ? 'text-white' : 'text-indigo-600',
                                'absolute inset-y-0 right-0 flex items-center pr-4'
                              )}
                            >
                              <CheckIcon className='h-5 w-5' aria-hidden='true' />
                            </span>
                          ) : null}
                        </>
                      )}
                    </Listbox.Option>
                  ))}
                </Listbox.Options>
              </Transition>
            </div>
          </>
        )}
      </Listbox>
    </>
  )
}

export default Select
