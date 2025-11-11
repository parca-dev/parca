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

import {useCallback, useState} from 'react';

import {Disclosure} from '@headlessui/react';
import {Icon} from '@iconify/react';
import cx from 'classnames';
import GitHubButton from 'react-github-btn';
import {usePopper} from 'react-popper';
import {Link, LinkProps, useLocation, useNavigate} from 'react-router-dom';

import {Button} from '@parca/components';
import {Parca, ParcaSmall} from '@parca/icons';
import {selectDarkMode, useAppSelector} from '@parca/store';
import {convertToQueryParams, parseParams} from '@parca/utilities';

import ReleaseNotesViewer from '../ReleaseNotesViewer';
import ThemeToggle from './ThemeToggle';

const links: {[path: string]: {label: string; href: string; external: boolean}} = {
  '/': {label: 'Explorer', href: `/`, external: false},
  '/compare': {label: 'Compare', href: 'compare', external: false},
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

  const queryParams = new URLSearchParams(location.search);
  const expressionA = queryParams.get('expression_a');
  const expressionB = queryParams.get('expression_b');
  const compareA = queryParams.get('compare_a');
  const compareB = queryParams.get('compare_b');

  const queryParamsURL = parseParams(window.location.search);

  /* eslint-disable @typescript-eslint/naming-convention */
  const {
    from_a,
    to_a,
    merge_from_a,
    merge_to_a,
    time_selection_a,
    dashboard_items,
    selection_a,
    expression_a,
  } = queryParamsURL;

  const isComparePage = expressionA !== null && expressionB !== null;

  const [compareHover, setCompareHover] = useState<boolean>(false);
  const [comparePopperReferenceElement, setComparePopperReferenceElement] =
    useState<HTMLDivElement | null>(null);
  const [comparePopperElement, setComparePopperElement] = useState<HTMLDivElement | null>(null);
  const {styles, attributes} = usePopper(comparePopperReferenceElement, comparePopperElement, {
    placement: 'bottom',
  });

  const compareExplanation =
    'Compare two profiles and see the relative difference between them more clearly.';

  const isCurrentPage = (item: {label: string; href: string; external: boolean}) => {
    const isCompareMode = compareA === 'true' || compareB === 'true';
    return (
      (item.href === 'compare' && (isComparePage || isCompareMode)) ||
      (!isComparePage && !isCompareMode && location.pathname === item.href)
    );
  };

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

  const queryToBePassed =
    expression_a === undefined
      ? {
          compare_a: 'true',
          compare_b: 'true',
        }
      : {
          compare_a: 'true',
          compare_b: 'true',
          dashboard_items: dashboard_items,
          expression_a: expression_a,
          expression_b: expression_a,
          from_a: from_a,
          from_b: from_a,
          to_a: to_a,
          to_b: to_a,
          time_selection_a: time_selection_a,
          time_selection_b: time_selection_a,
          selection_a: selection_a,
          merge_from_a: merge_from_a,
          merge_to_a: merge_to_a,
        };

  return (
    <Disclosure
      as="nav"
      className="relative z-10 bg-white dark:bg-gray-900 border-b-[1px] border-gray-300 dark:border-gray-700"
    >
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
                    <nav className="-mb-px flex space-x-4" aria-label="Tabs">
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
                              ? 'whitespace-nowrap text-sm font-medium bg-gray-200 text-gray-700 dark:bg-gray-800 dark:text-gray-200 py-2 px-4 rounded-lg'
                              : 'text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-100 whitespace-nowrap py-2 px-4 text-sm font-medium hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg'
                          ),
                          rel: item.external ? 'noreferrer' : undefined,
                        };
                        if (isCurrentPage(item)) {
                          props['aria-current'] = 'page';
                        }
                        return item.href === 'compare' ? (
                          <div ref={setComparePopperReferenceElement} key={item.label}>
                            <Button
                              className={cx(
                                isCurrentPage(item)
                                  ? 'whitespace-nowrap text-sm font-medium bg-gray-200 text-gray-700 dark:bg-gray-800 dark:text-gray-200 py-2 px-4 rounded-lg'
                                  : 'text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-100 whitespace-nowrap py-2 px-4 text-sm font-medium hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg',
                                'hover:no-underline focus:ring-0 focus:outline-none focus:ring-offset-0 h-full whitespace-nowrap font-medium'
                              )}
                              variant="link"
                              onClick={() =>
                                navigateTo(
                                  '/',
                                  {
                                    ...queryToBePassed,
                                  },
                                  {replace: true}
                                )
                              }
                              onMouseEnter={() => setCompareHover(true)}
                              onMouseLeave={() => setCompareHover(false)}
                              id="h-compare-button"
                            >
                              Compare
                            </Button>
                            {compareHover && (
                              <div
                                ref={setComparePopperElement}
                                style={styles.popper}
                                {...attributes.popper}
                                className="z-50"
                              >
                                <div className="flex">
                                  <div className="relative mx-2">
                                    <svg
                                      className="left-0 h-1 w-full text-black"
                                      x="0px"
                                      y="0px"
                                      viewBox="0 0 255 127.5"
                                    >
                                      <polygon
                                        className="fill-current"
                                        points="0,127.5 127.5,0 255,127.5"
                                      />
                                    </svg>
                                    <div className="right-0 w-40 rounded bg-black px-3 py-2 text-xs text-white">
                                      {compareExplanation}
                                    </div>
                                  </div>
                                </div>
                              </div>
                            )}
                          </div>
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
              <div className="flex items-center gap-3">
                <div className="text-sm font-medium text-gray-500 hover:border-gray-300 hover:text-gray-700 dark:text-gray-300">
                  <ReleaseNotesViewer version={window.APP_VERSION} />
                </div>
                <div className="hidden md:flex">
                  <GitHubStarButton />
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
