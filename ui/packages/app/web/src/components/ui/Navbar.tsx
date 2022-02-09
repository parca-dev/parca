import {Disclosure} from '@headlessui/react';
import {MenuIcon, XIcon} from '@heroicons/react/outline';
import {Parca, ParcaSmall} from '@parca/icons';
import cx from 'classnames';
import DarkModeToggle from './DarkModeToggle';

const links = [
  {name: 'Profiles', href: '/', current: true, external: false},
  {name: 'Help', href: 'https://parca.dev/docs/overview', current: false, external: true},
];

const Navbar = ({
  isDarkMode,
  setDarkMode,
}: {
  isDarkMode: boolean;
  setDarkMode: (mode: boolean) => void;
}) => {
  return (
    <Disclosure as="nav" className="dark:bg-gray-900 relative z-10">
      {({open}) => (
        <>
          <div className="mx-auto px-3">
            <div className="relative flex items-center justify-between h-16">
              <div className="absolute inset-y-0 left-0 flex items-center sm:hidden">
                {/* mobile menu button */}
                <Disclosure.Button className="inline-flex items-center justify-center p-2 rounded-md text-gray-400 hover:text-white hover:bg-gray-700 focus:outline-none focus:ring-2 focus:ring-inset focus:ring-white">
                  <span className="sr-only">Open main menu</span>
                  {open ? (
                    <XIcon className="block h-6 w-6" aria-hidden="true" />
                  ) : (
                    <MenuIcon className="block h-6 w-6" aria-hidden="true" />
                  )}
                </Disclosure.Button>
              </div>
              <div className="flex-1 flex items-center justify-center sm:items-stretch sm:justify-start">
                <div className="flex-shrink-0 flex items-center">
                  {/* image for small screens: */}
                  <div style={{padding: '5px'}} className="block lg:hidden h-8 w-auto rounded-full">
                    <ParcaSmall
                      style={{height: '100%', width: '100%'}}
                      className="block lg:hidden h-8 w-auto"
                    />
                  </div>
                  {/* image for larger screens: */}
                  <Parca
                    height={32}
                    style={{transform: 'translateY(5px)'}}
                    className="hidden lg:block h-8 w-auto"
                  />
                </div>
                <div className="hidden sm:block sm:ml-6">
                  <div className="flex space-x-4">
                    {links.map(item => (
                      <a
                        key={item.name}
                        href={item.href}
                        target={item.external ? '_blank' : undefined}
                        className={cx(
                          item.current
                            ? 'bg-gray-900 text-white'
                            : 'text-gray-700 dark:text-gray-300 hover:bg-gray-700 hover:text-white',
                          'px-3 py-2 rounded-md text-sm font-medium'
                        )}
                        aria-current={item.current ? 'page' : undefined}
                        rel="noreferrer"
                      >
                        {item.name}
                      </a>
                    ))}
                  </div>
                </div>
              </div>
              <div className="text-gray-500 dark:text-gray-0 absolute inset-y-0 right-0 flex items-center pr-2 sm:static sm:inset-auto sm:ml-6 sm:pr-0">
                {'{{.Version}}'}
              </div>
              <div className="text-gray-800 dark:text-gray-100 absolute inset-y-0 right-0 flex items-center pr-2 sm:static sm:inset-auto sm:ml-6 sm:pr-0">
                <a target="_blank" href="https://github.com/parca-dev/parca" rel="noreferrer">
                  GitHub
                </a>
              </div>
              <div className="text-gray-800 dark:text-gray-100 absolute inset-y-0 right-0 flex items-center pr-2 sm:static sm:inset-auto sm:ml-6 sm:pr-0">
                <DarkModeToggle isDarkMode={isDarkMode} setDarkMode={setDarkMode} />
              </div>
            </div>
          </div>

          <Disclosure.Panel className="sm:hidden">
            <div className="px-2 pt-2 pb-3 space-y-1">
              {links.map(item => (
                <a
                  key={item.name}
                  href={item.href}
                  className={cx(
                    item.current
                      ? 'bg-gray-900 text-white'
                      : 'text-gray-300 hover:bg-gray-700 hover:text-white',
                    'block px-3 py-2 rounded-md text-base font-medium'
                  )}
                  aria-current={item.current ? 'page' : undefined}
                >
                  {item.name}
                </a>
              ))}
            </div>
          </Disclosure.Panel>
        </>
      )}
    </Disclosure>
  );
};

export default Navbar;
