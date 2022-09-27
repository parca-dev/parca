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

import React from 'react';

interface EmptyStateProps {
  isEmpty: boolean;
  body?: string | JSX.Element | JSX.Element[];
  title?: string;
  icon?: SVGElement;
  children: JSX.Element | JSX.Element[];
}

const DEFAULT_ICON = (
  <svg width="64" height="41" viewBox="0 0 64 41" xmlns="http://www.w3.org/2000/svg">
    <g transform="translate(0 1)" fill="none" fillRule="evenodd">
      <ellipse fill="#F5F5F5" cx="32" cy="33" rx="32" ry="7"></ellipse>
      <g fillRule="nonzero" stroke="#D9D9D9">
        <path d="M55 12.76L44.854 1.258C44.367.474 43.656 0 42.907 0H21.093c-.749 0-1.46.474-1.947 1.257L9 12.761V22h46v-9.24z"></path>
        <path
          d="M41.613 15.931c0-1.605.994-2.93 2.227-2.931H55v18.137C55 33.26 53.68 35 52.05 35h-40.1C10.32 35 9 33.259 9 31.137V13h11.16c1.233 0 2.227 1.323 2.227 2.928v.022c0 1.605 1.005 2.901 2.237 2.901h14.752c1.232 0 2.237-1.308 2.237-2.913v-.007z"
          fill="#FAFAFA"
        ></path>
      </g>
    </g>
  </svg>
);

const EmptyState = ({title, icon, body, isEmpty, children}: EmptyStateProps): JSX.Element => {
  return isEmpty ? (
    <div className="flex justify-center items-center flex-col h-64">
      <>
        {icon ?? DEFAULT_ICON}
        <p className="flex items-center justify-center text-xl p-4 text-gray-500">
          {title ?? 'No data available'}
        </p>
        {Boolean(body) && (
          <div className="flex items-center justify-center p-1 text-gray-500 text-sm">{body}</div>
        )}
      </>
    </div>
  ) : (
    <>{children}</>
  );
};

export default EmptyState;
