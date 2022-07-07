import {Fragment} from 'react';

import {Dialog, Transition} from '@headlessui/react';
import cx from 'classnames';

const Modal = ({
  isOpen,
  closeModal,
  title,
  children,
  className = 'max-w-md',
}: {
  isOpen: boolean;
  title: string;
  closeModal: () => void;
  children: React.ReactNode;
  className?: string;
}) => {
  return (
    <>
      <Transition appear show={isOpen} as={Fragment}>
        <Dialog
          as="div"
          className="fixed inset-0 z-10 overflow-y-auto bg-black/60"
          onClose={closeModal}
        >
          <div className="min-h-screen px-4 text-center">
            <Transition.Child
              as={Fragment}
              enter="ease-out duration-300"
              enterFrom="opacity-0"
              enterTo="opacity-100"
              leave="ease-in duration-200"
              leaveFrom="opacity-100"
              leaveTo="opacity-0"
            >
              <Dialog.Overlay className="fixed inset-0" />
            </Transition.Child>

            {/* This element is to trick the browser into centering the modal contents. */}
            <span className="inline-block h-screen align-middle" aria-hidden="true">
              &#8203;
            </span>
            <Transition.Child
              as={Fragment}
              enter="ease-out duration-300"
              enterFrom="opacity-0 scale-95"
              enterTo="opacity-100 scale-100"
              leave="ease-in duration-200"
              leaveFrom="opacity-100 scale-100"
              leaveTo="opacity-0 scale-95"
            >
              <div
                className={cx(
                  'inline-block w-fit min-w-md p-6 my-8 overflow-hidden text-left align-middle transition-all transform bg-white dark:bg-gray-700 shadow-xl rounded-2xl',
                  {[className]: className?.length > 0}
                )}
              >
                <Dialog.Title as="h3" className="text-lg font-medium leading-6">
                  {title}
                </Dialog.Title>
                {children}
              </div>
            </Transition.Child>
          </div>
        </Dialog>
      </Transition>
    </>
  );
};

export default Modal;
