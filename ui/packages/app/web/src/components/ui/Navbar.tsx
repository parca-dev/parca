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

import {Disclosure} from '@headlessui/react';
import {Icon} from '@iconify/react';
import cx from 'classnames';
import GitHubButton from 'react-github-btn';

import {UserPreferences} from '@parca/components';
import {Parca, ParcaSmall} from '@parca/icons';
import {selectDarkMode, useAppSelector} from '@parca/store';

import ReleaseNotesViewer from '../ReleaseNotesViewer';
import DarkModeToggle from './DarkModeToggle';

const pathPrefix = process.env.NODE_ENV === 'development' ? '' : window.PATH_PREFIX;

const links: {[path: string]: {label: string; href: string; external: boolean}} = {
  '/': {label: 'Profiles', href: `${pathPrefix}/`, external: false},
  '/targets': {label: 'Targets', href: `${pathPrefix}/targets`, external: false},
  '/help': {label: 'Help', href: 'https://parca.dev/docs/overview', external: true},
};

const GitHubStarButton = () => {
  const isDarkMode = useAppSelector(selectDarkMode);

  return (
    <GitHubButton
      href="https://github.com/parca-dev/parca"
      data-color-scheme={isDarkMode ? 'dark' : 'light'}
      data-size="large"
      data-show-count="false"
      aria-label="Star parca-dev/parca on GitHub"
    >
      Star
    </GitHubButton>
  );
};

const Navbar = () => {
  const removePathPrefix = (href: string) =>
    href.startsWith(window.PATH_PREFIX) ? href.slice(window.PATH_PREFIX.length) : href;

  const getPageByHref = (href: string = '/'): {label: string; href: string; external: boolean} => {
    const link = removePathPrefix(href);
    return links[link] ?? links['/'];
  };

  const currentPage = getPageByHref(window.location.pathname);
  const isCurrentPage = (item: {label: string; href: string; external: boolean}) =>
    item.href === currentPage.href;

  return (
    <Disclosure as="nav" className="dark:bg-gray-900 relative z-10">
      {({open}) => (
        <>
          <div className="mx-auto px-3">
            <div className="relative flex items-center justify-between h-16 gap-1">
              <div className="absolute inset-y-0 left-0 flex items-center sm:hidden">
                {/* mobile menu button */}
                <Disclosure.Button className="inline-flex items-center justify-center p-2 rounded-md text-gray-400 hover:text-white hover:bg-gray-700 focus:outline-none focus:ring-2 focus:ring-inset focus:ring-white">
                  <span className="sr-only">Open main menu</span>
                  {open ? (
                    <Icon icon="heroicons:x-mark-20-solid" aria-hidden="true" />
                  ) : (
                    <Icon icon="heroicons:bars-3-20-solid" aria-hidden="true" />
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
                  <div className="flex gap-2 items-center">
                    {Object.values(links).map(item => (
                      <a
                        key={item.label}
                        href={item.href}
                        target={item.external ? '_blank' : undefined}
                        className={cx(
                          isCurrentPage(item)
                            ? 'bg-gray-900 dark:bg-gray-700 text-white'
                            : 'text-gray-700 dark:text-gray-300 hover:bg-gray-700 hover:text-white',
                          'px-3 py-2 rounded-md text-sm font-medium'
                        )}
                        aria-current={isCurrentPage(item) ? 'page' : undefined}
                        rel="noreferrer"
                      >
                        {item.label}
                      </a>
                    ))}
                    <div className="px-3 hidden md:flex pt-2">
                      <GitHubStarButton />
                    </div>
                  </div>
                </div>
              </div>
              <div className="flex gap-3 items-center">
                <div className="text-gray-500">
                  <ReleaseNotesViewer version={window.APP_VERSION} />
                </div>
                <div className="pl-4">
                  <a target="_blank" href="https://github.com/parca-dev/parca" rel="noreferrer">
                    GitHub
                  </a>
                </div>
                <div className="">
                  <UserPreferences />
                </div>
                <div className="">
                  <DarkModeToggle />
                </div>
              </div>
            </div>
          </div>

          <Disclosure.Panel className="sm:hidden">
            <div className="px-2 pt-2 pb-3 space-y-1">
              {Object.values(links).map(item => (
                <a
                  key={item.label}
                  href={item.href}
                  className={cx(
                    isCurrentPage(item)
                      ? 'bg-gray-900 dark:bg-gray-700 text-white'
                      : 'text-gray-700 dark:text-gray-300 hover:bg-gray-700 hover:text-white',
                    'block px-3 py-2 rounded-md text-base font-medium'
                  )}
                  aria-current={isCurrentPage(item) ? 'page' : undefined}
                >
                  {item.label}
                </a>
              ))}
              <div className="px-3 pt-1">
                <GitHubStarButton />
              </div>
            </div>
          </Disclosure.Panel>
        </>
      )}
    </Disclosure>
  );
};

export default Navbar;
