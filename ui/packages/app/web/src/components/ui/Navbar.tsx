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

import {useCallback} from 'react';

import {Disclosure} from '@headlessui/react';
import {Icon} from '@iconify/react';
import cx from 'classnames';
import GitHubButton from 'react-github-btn';
import {Link, LinkProps, useLocation, useNavigate} from 'react-router-dom';

import {Button} from '@parca/components';
import {Parca, ParcaSmall} from '@parca/icons';
import {compareProfile} from '@parca/profile';
import {selectDarkMode, useAppSelector} from '@parca/store';
import {convertToQueryParams} from '@parca/utilities';

import ReleaseNotesViewer from '../ReleaseNotesViewer';
import ThemeToggle from './ThemeToggle';

const links: {[path: string]: {label: string; href: string; external: boolean}} = {
  '/': {label: 'Explorer', href: `/`, external: false},
  '/compare': {label: 'Compare', href: 'onclick', external: false},
  '/targets': {label: 'Targets', href: `/targets`, external: false},
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
  const location = useLocation();
  const navigate = useNavigate();

  const isCurrentPage = (item: {label: string; href: string; external: boolean}) =>
    location.pathname === item.href;

  const navigateTo = useCallback(
    (path: string, queryParams: any, options?: {replace?: boolean}) => {
      navigate(
        {
          pathname: path,
          search: `?${convertToQueryParams(queryParams)}`,
        },
        options ?? {}
      );
    },
    [navigate]
  );

  return (
    <Disclosure as="nav" className="relative z-10 dark:bg-gray-900">
      {({open}) => (
        <>
          <div className="mx-auto px-3">
            <div className="relative flex h-16 items-center justify-between gap-1">
              <div className="absolute inset-y-0 left-0 flex items-center sm:hidden">
                {/* mobile menu button */}
                <Disclosure.Button className="inline-flex items-center justify-center rounded-md p-2 text-gray-400 hover:bg-gray-700 hover:text-white focus:outline-none focus:ring-2 focus:ring-inset focus:ring-white">
                  <span className="sr-only">Open main menu</span>
                  {open ? (
                    <Icon icon="heroicons:x-mark-20-solid" aria-hidden="true" />
                  ) : (
                    <Icon icon="heroicons:bars-3-20-solid" aria-hidden="true" />
                  )}
                </Disclosure.Button>
              </div>
              <div className="flex flex-1 items-center justify-center sm:items-stretch sm:justify-start">
                <div className="flex flex-shrink-0 items-center">
                  {/* image for small screens: */}
                  <div style={{padding: '5px'}} className="block h-8 w-auto rounded-full lg:hidden">
                    <ParcaSmall
                      style={{height: '100%', width: '100%'}}
                      className="block h-8 w-auto lg:hidden"
                    />
                  </div>
                  {/* image for larger screens: */}
                  <Parca
                    height={32}
                    style={{transform: 'translateY(5px)'}}
                    className="hidden h-8 w-auto lg:block"
                  />
                </div>
                <div className="hidden sm:ml-6 sm:block">
                  <div className="flex items-center gap-2">
                    <div className="border-b border-gray-200">
                      <nav className="-mb-px flex space-x-8" aria-label="Tabs">
                        {Object.values(links).map(item => {
                          const href = item.href;
                          const props: {
                            target?: LinkProps['target'];
                            className: string;
                            rel?: LinkProps['rel'];
                            'aria-current'?: 'page';
                          } = {
                            target: item.external ? '_blank' : undefined,
                            className: cx(
                              isCurrentPage(item)
                                ? 'border-indigo-500 dark:border-gray-100 text-indigo-600 dark:text-gray-100 whitespace-nowrap border-b-2 py-4 px-1 text-sm font-medium'
                                : 'border-transparent text-gray-500 dark:text-gray-400 hover:border-gray-300 hover:text-gray-700 dark:hover:text-gray-100 whitespace-nowrap border-b-2 py-4 px-1 text-sm font-medium'
                            ),
                            rel: item.external ? 'noreferrer' : undefined,
                          };
                          if (isCurrentPage(item)) {
                            props['aria-current'] = 'page';
                          }
                          return item.href === 'onclick' ? (
                            <Button
                              className="hover:no-underline rounded-none hover:border-gray-300 border-transparent border-b-2 hover:text-gray-700 dark:hover:text-gray-100 focus:ring-0 focus:outline-none focus:ring-offset-0"
                              variant="link"
                              onClick={() => compareProfile(navigateTo)}
                            >
                              Compare
                            </Button>
                          ) : item.external === true ? (
                            <a key={item.label} {...props} href={href}>
                              {item.label}
                            </a>
                          ) : (
                            <Link key={item.label} {...props} to={href}>
                              {item.label}
                            </Link>
                          );
                        })}
                      </nav>
                    </div>
                  </div>
                </div>
              </div>
              <div className="flex items-center gap-3">
                <div className="text-sm font-medium text-gray-500 hover:border-gray-300 hover:text-gray-700 dark:text-gray-300">
                  <ReleaseNotesViewer version={window.APP_VERSION} />
                </div>
                <div className="hidden md:flex">
                  <GitHubStarButton />
                </div>
                <div className="text-sm font-medium text-gray-500 hover:border-gray-300 hover:text-gray-700 dark:text-gray-300">
                  <a target="_blank" href="https://github.com/parca-dev/parca" rel="noreferrer">
                    GitHub
                  </a>
                </div>
                <div className="text-sm font-medium text-gray-500 hover:border-gray-300 hover:text-gray-700">
                  <ThemeToggle />
                </div>
              </div>
            </div>
          </div>

          <Disclosure.Panel className="sm:hidden">
            <div className="space-y-1 px-2 pb-3 pt-2">
              {Object.values(links).map(item => (
                <a
                  key={item.label}
                  href={item.href}
                  className={cx(
                    isCurrentPage(item)
                      ? 'bg-gray-900 text-white dark:bg-gray-700'
                      : 'text-gray-700 hover:bg-gray-700 hover:text-white dark:text-gray-300',
                    'block rounded-md px-3 py-2 text-base font-medium'
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
