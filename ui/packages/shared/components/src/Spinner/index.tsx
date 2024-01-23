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

export const MS = () => (
  <svg
    fill="none"
    height="452"
    viewBox="0 0 1435 452"
    width="100%"
    xmlns="http://www.w3.org/2000/svg"
  >
    <defs>
      <linearGradient id="y-chart-shimmer" x1="0%" y1="0%" x2="0%" y2="100%">
        <stop offset="0%" style={{stopColor: '#ebebeb', stopOpacity: 1}} />
        <stop offset="50%" style={{stopColor: '#F6F6F6', stopOpacity: 1}}>
          <animate
            attributeName="offset"
            values="-2; -2; 1"
            keyTimes="0; 0.25; 1"
            dur="2s"
            repeatCount="indefinite"
          ></animate>
        </stop>
        <stop offset="100%" style={{stopColor: '#ebebeb', stopOpacity: 1}} />
      </linearGradient>
      <linearGradient id="x-chart-shimmer" x1="0%" y1="0%" x2="100%" y2="0%">
        <stop offset="0.599964" stop-color="#f3f3f3" stop-opacity="1">
          <animate
            attributeName="offset"
            values="-2; -2; 1"
            keyTimes="0; 0.25; 1"
            dur="2s"
            repeatCount="indefinite"
          ></animate>
        </stop>
        <stop offset="1.59996" stop-color="#ecebeb" stop-opacity="1">
          <animate
            attributeName="offset"
            values="-1; -1; 2"
            keyTimes="0; 0.25; 1"
            dur="2s"
            repeatCount="indefinite"
          ></animate>
        </stop>
        <stop offset="2.59996" stop-color="#f3f3f3" stop-opacity="1">
          <animate
            attributeName="offset"
            values="0; 0; 3"
            keyTimes="0; 0.25; 1"
            dur="2s"
            repeatCount="indefinite"
          ></animate>
        </stop>
      </linearGradient>
    </defs>

    <path d="m3.5 146h19v111h-19z" fill="url(#y-chart-shimmer)" />
    <g stroke="#ececec">
      <path d="m53 19h1378v365h-1378z" />
      <path d="m52.5 139.039h1379" />
      <path d="m52.5 79.8652h1379" />
      <path d="m52.5 198.213h1379" />
      <path d="m52.5 257.387h1379" />
      <path d="m52.5 316.561h1379" />
      <path d="m284.412 18.5v366" />
      <path d="m512.765 18.5v366" />
      <path d="m739.322 18.5v366" />
      <path d="m967.669 18.5v366" />
      <path d="m1196.01 18.5v366" />
    </g>
    <path
      id="line-chart"
      d="m59.0469 219.018 15.3287 18.443 15.3521-9.261 15.3053-45.218 15.346 19.161 15.475 26.026 15.177-8.714 15.352 24.135 15.323-11.89 15.335-17.409 15.358 13.334 15.299-14.885 15.34 5.708 15.329 9.041 15.341 1.306 15.328-8.993 15.335 15.737 15.329 1.446 15.475-50.644 15.223 15.875 15.323 19.059 15.305-.505 15.335 18.037 15.34-10.759 15.329-13.82 15.335 19.318 15.404-20.67 15.3 4.73 15.305 6.21 15.352 5.957 15.323-10.731 15.487 23.078 15.171-15.997 15.328-41.612 15.341 19.894 15.328 28.445 15.323-11.888 15.358 18.418 15.317-15.533 15.335-5.416 15.358 14.761 15.311-17.545 15.358-.133 15.305 19.095 15.481-2.97 15.177-15.239 15.815 13.635 15.088 3.429 15.124-50.091 15.329 23.347 15.34 18.612 15.335-6.35 15.311 20.201 15.334-5.183 15.335-21.711 15.346 22.854 15.358-25.831 15.423 9.051 15.206 7.617 15.352 5.504 15.317-12.49 15.323 20.007 15.347-1.744 15.32-54.27 15.34 20.107 15.33 26.672 15.33-10.381 15.33 21.335 15.34-11.158 15.39-10.58 15.39 14.077 15.21-15.575 15.34 2.279 15.33 11.167 15.34-1.543 15.32-9.005 15.34 18.423 15.34-4.802 15.34-49.412 15.32 21.411 15.35 22.053 15.32-9.192 15.45 20.823 15.25-9.207 15.29-9.304 15.35 11.078 15.32-16.693 15.33 2.881 15.34 13.074 15.33 3.252"
      stroke="#cecbcb"
      stroke-width="1.00044"
      // fill="url(#shimmerGradient)"
    />
    <path
      id="line-chart"
      d="m58.9473 245.545 15.3521-3.548 15.2995-10.784 15.4981-31.534 15.218 20.347 15.288 11.129 15.358 19.748 15.346 1.114 15.311-25.752 15.329 18.418 15.358-20.204 15.317 2.293 15.305 17.368 15.335-5.174 15.334-9.743 15.347 19.442 15.44.767 15.264-18.629 15.282-31.338 15.375 16.618 15.388 16.13 15.229 12.922 15.352 5.358 15.335-25.327 15.328 24.048 15.388-22.26 15.322-1.878 15.306 17.355 15.334-7.564 15.516-7.18 15.13 22.022 15.323-6.657 15.557-15.232 15.124-26.918 15.334 19.828 15.352 15.123 15.323 11 15.341 2.841 15.328-23.562 15.335 17.804 15.323-16.64 15.393-.623 15.428 18.277 15.218-5.88 15.276-5.341 15.733 5.15 14.995 5.253 15.317-16.748 15.334-22.767 15.352 10.862 15.37 14.608 15.417 14.536 15.147 7.289 15.393-22.514 15.411 14.514 15.293-23.282 15.253 7.477 15.528 14.062 15.246-6.945 15.376-6.239 15.194 19.758 15.311-3.22 15.343-15.686 15.37-27.035 15.36 11.388 15.29 19.319 15.31 14.414 15.49-1.095 15.19-18.602 15.35 19.75 15.31-19.083 15.44-1.349 15.24 11.252 15.33-2.912 15.34-8.1 15.35 15.276 15.82 2.633 14.84-10.049 15.41 13.715 15.33 5.569 15.46-34.366 15.08 36.473 15.38-11.922 15.35-33.985 15.39-2.682 15.34-30.202 15.23 21.728 15.33 43.202 15.31-3.283 15.51-16.943"
      stroke="#cecbcb"
      stroke-width="1.00044"
    />
    <path
      id="line-chart"
      d="m59.0469 376.57 15.3287.577 15.3521 3.072 15.3053-1.15 15.346-1.526 15.475 1.109 15.177-.92 15.352-1.553 15.323 1.931 15.335-2.873 15.358 3.053 15.299-.565 15.34.764 15.329.966 15.341-.194 15.328-1.92 15.335 1.92 15.329-2.471 15.475 3.415 15.223-1.141 15.323-.961 15.305 1.541h15.335l15.34-3.652 15.329.007 15.335 2.893 15.404-4.079 15.3 4.637 15.305-1.531 15.352-1.354 15.323.225 15.487 2.636 15.171-2.085 15.328.575 15.341.765 15.328-.959 15.323-.381 15.358-.966 15.317 2.502 15.335-1.152 15.358.567 15.311-1.519 15.358-2.902 15.305 6.159 15.481-2.543 15.177 1.235 15.815-1.854 15.088.398 15.124-.308 15.329.187 15.34.965 15.335-.201 15.311 1.354 15.334-1.347 15.335-.577 15.346-.185 15.358.005 15.423-.242 15.206-.532 15.352 2.106 15.317-3.077 15.323 3.082 15.347-.005 15.32-1.15 15.34 1.534 15.33-.384 15.33-2.494 15.33.381 15.34-.386 15.39-.206 15.39 1.325 15.21 1.383 15.34-.003 15.33-.769 15.34.959 15.32.196 15.34-.388 15.34-2.106 15.34 3.069 15.32-1.533 15.35.953 15.32-.747 15.45-.437 15.25.6 15.29.781 15.35-1.347 15.32.388 15.33.576 15.34-2.878 15.33.381"
      stroke="#cecbcb"
      stroke-width="1.00044"
    />
    <path
      id="line-chart"
      d="m59.0469 374.136 15.3287.577 15.3521 3.073 15.3053-1.151 15.346-1.526 15.475 1.109 15.177-.92 15.352-1.553 15.323 1.932 23.589-28.434 7.104 28.613 15.299-.565 15.34.764 15.329.966 15.341-.194 15.328-13.157 15.335 13.157 15.329-13.708 15.475 14.652 15.223-1.141 15.323-.961 15.305 1.541h15.335l15.34-3.652 15.329.008 15.335 2.892 15.404-4.079 15.3 4.637 15.305-1.531 15.352-1.354 36.469-11.019-5.659 13.88 15.171-2.084 15.328.575 15.341.764 15.328-.958 15.323-.381 15.358-.966 15.317 2.502 15.335-1.153 15.358-12.179 15.311 11.228 15.358-20.185h20.197 17.298 24.283l15.088 8.957 8.678 4.07 6.446 7.342 15.329.187 15.34.966 15.335-.202 15.311 1.354 15.335-1.346 15.334-.578 15.346-.184 15.358.005 15.423-.243 15.206-.532 15.352 2.107 53.597-21.903-22.957 21.908 15.347-.005 15.32-1.15 15.34 1.533 15.33-.383 15.33-2.495 11.64-6.381 19.03 6.376 15.39-.206 15.39 1.325 15.21 1.383 23.87-8.878 6.8 8.107 15.34.958 15.32.197 15.34-.389 15.34-2.106 15.34 3.07 12.13-13.907 18.54 13.327 15.32-.748 15.45-.436 15.25.599 30.64-68.106v67.541l15.32.388 15.33.575 15.34-2.878 15.33.381"
      stroke="#cecbcb"
      stroke-width="1.00044"
    />
    <path
      id="line-chart"
      d="m59.4253 311.341 15.3639-31.823 15.3111-8.503 15.3467 44.655 15.305-32.747 15.375 4.809 15.306 30.68 15.317-59.164 15.358 52.823 15.323-26.855 15.328 27.841 15.329-24.929 15.341-4.672 15.322 22.522 15.423-44.558 15.329 29.518 15.246.893 15.353 17.938 15.328-30.634 15.335 28.607 15.34-16.028 15.323-7.496 15.358 5.934 15.323-.148 15.323 5.552 15.323-.949 15.475 8.971 15.182-22.706 15.341-.095 15.328 24.82 15.335-23.647 15.329 7.546 15.34 1.228 15.358 24.572 15.305-99.189 15.388 102.745 15.276-34.185 15.34 35.483 15.329-54.983 15.446 18.656 15.217 25.016 15.335-17.115 15.358-29.763 15.34 57.72 15.294-19.619 15.346-5.962 15.335 4.516 15.364 14.094 15.317-33.665 15.352 24.837 15.352 7.248 15.276-21.143 15.329 6.236 15.346 15.096 15.323.386 15.44-35.985 15.223 32.561 15.335-17.486 15.328 24.829 15.353-21.43 15.317 13.483 15.334-13.638 15.339-5.36 15.33 12.625 15.36-15.465 15.32.92 15.32 27.657 15.33-32.297 15.36 36.252 15.31-20.852 15.33-7.122 15.34-11.095 15.33 33.376 15.32-24.929 15.34 17.625 15.33-41.721 15.35 22.521 15.44 14.635 15.21-10.153 15.35-12.878 15.33 36.424 15.38-20.165 15.3-10.066 15.34 11.725 15.32-10.225 15.33 7.146 15.32-1.029 15.38 18.261 15.3-20.819 15.33-27.061"
      stroke="#cecbcb"
      stroke-width="2.00088"
    />
    <path
      id="line-chart"
      d="m58.9473 259.676 15.3521 3.388 15.2995-10.393 15.4981 18.792 15.218 11.152 15.288-17.566 15.358 8.583 15.346-8.632 15.311 1.415 15.329 8.28 15.358-14.242 15.317-4.472 15.305 6.867 15.335 12.094 15.334-28.236 15.347 1.767 15.44 8.362 15.264 17.11 15.282-11.155 15.375 8.563 15.388 5.256 15.229-24.921 15.352-9.962 15.335 23.993 15.328-1.32 15.388-11.609 15.322 13.349 15.306-1.726 15.334-26.542 15.516 25.368 15.13 8.937 15.323-2.034 15.557-14.817 15.124 25.017 15.334-57.563 15.352 69.412 15.323-36.021 15.341 7.103 15.328 12.376 15.335-40.159 15.323 14.883 15.393 24.569 15.428-21.667 15.218 30.112 15.276-16.528 15.733 20.75 14.995-27.918 15.317 7.226 15.334-8.085 15.352 24.905 15.37-32.445 15.417 23.631 15.147-20.08 15.393-.187 15.411 38.217 15.293-30.714 15.253 12.839 15.528-29.234 15.246 31.882 15.376-30.976 15.194 27.814 15.311-19.879 15.343 24.747 15.37-31.188 15.36 15.579 15.29-.269 15.31-10.182 15.49 25.827 15.19-42.428 15.35 42.767 15.31-16.105 15.44 9.709 15.24-13.041 15.33 20.83 15.34-51.452 15.35 50.853 15.82-16.562 14.84-29.219 15.41 25.322 15.33-7.421 15.46 22.483 15.08-2.201 15.38-24.99 15.35 19.702 15.39-8.27 15.34 6.186 15.23-4.62 15.33 10.512 15.31-14.436 15.51-40.224"
      stroke="#cecbcb"
      stroke-width="1.00044"
    />
    <path
      id="line-chart"
      d="m59.4258 230.887 15.3638-21.474 15.3112 8.557 15.3462-18.572 15.305-6.134 15.376 10.155 15.305 9.855 15.317-7.108 15.358 3.269 15.323-16.506 15.329 22.153 15.329-33.546 15.34 31.983 15.323-8.265 15.422-1.92 15.329 16.405 15.247-28.742 15.352 20.399 15.329-28.741 15.334 26.511 15.341-11.969 15.323-.592 15.358 4.788 15.323 5.363 15.322 4.419 15.323-4.861 15.475 8.506 15.183-19.127 15.34 20.335 15.329-20.318 15.334 26.091 15.329-25.666 15.341 1.907 15.358-12.347 15.305-56.281 15.387 74.052 15.276 5.066 15.341 12.306 15.328-46.621 15.446 53.513 15.218-23.379 15.334 23.704 15.358-35.602 15.341 7.251 15.293-.099 15.347 8.942 15.334 20.743 15.364-42.153 15.317 17.498 15.352 9.68 15.352-.65 15.277 16.365 15.328-23.694 15.346-2.028 15.323-17.569 15.44 35.608 15.224-30.85 15.334 27.785 15.329-15.042 15.352-11.146 15.317 23.165 15.335 1.878 15.338-17.29 15.33 6.423 15.36 1.345 15.32-3.939 15.32-8.643 15.33 19.811 15.36-22.464 15.31 26.482 15.33-46.596 15.34 27.785 15.33-9.447 15.32 15.79 15.34 18.984 15.33-28.761 15.35 11.748 15.44-9.758 15.21 17.445 15.35-13.349 15.33-19.197 15.38 44.466 15.3-34.757 15.34 2.291 15.32 14.713 15.33-20.826 15.32 27.56 15.38-4.55 15.3 4.938 15.33-23.451"
      stroke="#cecbcb"
      stroke-width="1.00044"
    />
    <path id="x-shimmer-chart" d="m635 413.5h165v19h-165z" fill="url(#x-chart-shimmer)" />
  </svg>
);

<svg
  fill="none"
  height="570"
  viewBox="0 0 1415 570"
  width="1415"
  xmlns="http://www.w3.org/2000/svg"
>
  <g fill="#ebebeb">
    <path d="m1 101.703h19v26h-19z" />
    <path d="m51 101.703h254v26h-254z" />
    <path d="m315 101.703h99v26h-99z" />
    <path d="m420 101.703h41v26h-41z" />
    <path d="m468 101.703h260v26h-260z" />
    <path d="m734 101.703h104v26h-104z" />
    <path d="m843 101.703h11v26h-11z" />
    <path d="m863 101.703h73v26h-73z" />
    <path d="m1077 101.703h63v26h-63z" />
    <path d="m1007 101.703h70v26h-70z" />
    <path d="m954 102h46v26h-46z" />
    <path d="m1144 102h10v26h-10z" />
    <path d="m1159 101.703h24v26h-24z" />
    <path d="m1189 101.703h80v26h-80z" />
    <path d="m1274 101.703h139v26h-139z" />
    <path d="m1 135.703h19v26h-19z" />
    <path d="m51 135.703h218v26h-218z" />
    <path d="m274 135.703h25v26h-25z" />
    <path d="m315 135.703h85v26h-85z" />
    <path d="m441 135.703h8v26h-8z" />
    <path d="m453 135.703h8v26h-8z" />
    <path d="m405 135.703h8v26h-8z" />
    <path d="m417 135.703h8v26h-8z" />
    <path d="m429 135.703h8v26h-8z" />
    <path d="m468 135.703h138v26h-138z" />
    <path d="m613 135.703h115v26h-115z" />
    <path d="m734 135.703h61v26h-61z" />
    <path d="m800 135.703h38v26h-38z" />
    <path d="m843 135.703h7v26h-7z" />
    <path d="m863 135.703h28v26h-28z" />
    <path d="m899 135.703h33v26h-33z" />
    <path d="m1006 136h134v26h-134z" />
    <path d="m954 135.703h16v26h-16z" />
    <path d="m973 135.703h27v26h-27z" />
    <path d="m1144 136h10v26h-10z" />
    <path d="m1157 135.703h26v26h-26z" />
    <path d="m1189 135.703h80v26h-80z" />
    <path d="m1274 135.703h139v26h-139z" />
    <path d="m1 169.703h19v26h-19z" />
    <path d="m51 169.703h50v26h-50z" />
    <path d="m111 169.703h143v26h-143z" />
    <path d="m260 169.703h35v26h-35z" />
    <path d="m320 169.703h54v26h-54z" />
    <path d="m315 169.703h2v26h-2z" />
    <path d="m468 169.703h108v26h-108z" />
    <path d="m580 169.703h20v26h-20z" />
    <path d="m613 169.703h58v26h-58z" />
    <path d="m684 169.703h44v26h-44z" />
    <path d="m737 169.703h31v26h-31z" />
    <path d="m771 169.703h22v26h-22z" />
    <path d="m800 169.703h23v26h-23z" />
    <path d="m827 169.703h6v26h-6z" />
    <path d="m675 169.703h5v26h-5z" />
    <path d="m843 169.703h7v26h-7z" />
    <path d="m863 169.703h22v26h-22z" />
    <path d="m1006 169.703h46v26h-46z" />
    <path d="m1057 169.703h83v26h-83z" />
    <path d="m957 169.703h10v26h-10z" />
    <path d="m973 169.703h27v26h-27z" />
    <path d="m1144 170h10v26h-10z" />
    <path d="m1157 169.703h7v26h-7z" />
    <path d="m1168 169.703h7v26h-7z" />
    <path d="m1193 169.703h54v26h-54z" />
    <path d="m1274 169.703h54v26h-54z" />
    <path d="m1333 169.703h54v26h-54z" />
    <path d="m1392 169.703h7v26h-7z" />
    <path d="m1403 169.703h7v26h-7z" />
    <path d="m1256 169.703h13v26h-13z" />
    <path d="m1 203.703h19v26h-19z" />
    <path d="m51 203.703h141v26h-141z" />
    <path d="m198 203.703h94v26h-94z" />
    <path d="m320 203.703h55v26h-55z" />
    <path d="m315 203.703h2v26h-2z" />
    <path d="m468 203.703h64v26h-64z" />
    <path d="m613 203.703h58v26h-58z" />
    <path d="m684 203.703h40v26h-40z" />
    <path d="m675 203.703h5v26h-5z" />
    <path d="m537 203.703h39v26h-39z" />
    <path d="m581 203.703h9v26h-9z" />
    <path d="m843 203.703h5v26h-5z" />
    <path d="m863 203.703h15v26h-15z" />
    <path d="m1100 203.703h40v26h-40z" />
    <path d="m1057 203.703h40v26h-40z" />
    <path d="m1006 203.703h46v26h-46z" />
    <path d="m959 203.703h8v26h-8z" />
    <path d="m979 203.703h21v26h-21z" />
    <path d="m1144 204h10v26h-10z" />
    <path d="m1168 203.703h7v26h-7z" />
    <path d="m1 237.703h19v26h-19z" />
    <path d="m51 237.703h104v26h-104z" />
    <path d="m198 237.703h90v26h-90z" />
    <path d="m160 237.703h19v26h-19z" />
    <path d="m320 237.703h55v26h-55z" />
    <path d="m315 237.703h2v26h-2z" />
    <path d="m468 237.703h59v26h-59z" />
    <path d="m537 237.703h32v26h-32z" />
    <path d="m613 237.703h58v26h-58z" />
    <path d="m684 237.703h36v26h-36z" />
    <path d="m675 237.703h5v26h-5z" />
    <path d="m843 237.703h3v26h-3z" />
    <path d="m863 237.703h8v26h-8z" />
    <path d="m1006 237.703h46v26h-46z" />
    <path d="m1110 237.703h30v26h-30z" />
    <path d="m1057 237.703h30v26h-30z" />
    <path d="m1090 237.703h7v26h-7z" />
    <path d="m1090 237.703h7v26h-7z" />
    <path d="m1100 237.703h8v26h-8z" />
    <path d="m961 237.703h6v26h-6z" />
    <path d="m979 237.703h21v26h-21z" />
    <path d="m1144 238h10v26h-10z" />
    <path d="m1 271.703h19v26h-19z" />
    <path d="m51 271.703h128v26h-128z" />
    <path d="m198 271.703h84v26h-84z" />
    <path d="m320 271.703h55v26h-55z" />
    <path d="m315 271.703h2v26h-2z" />
    <path d="m468 271.703h50v26h-50z" />
    <path d="m537 271.703h21v26h-21z" />
    <path d="m613 271.703h58v26h-58z" />
    <path d="m684 271.703h29v26h-29z" />
    <path d="m675 271.703h5v26h-5z" />
    <path d="m843 271.703h1v26h-1z" />
    <path d="m863 271.703h4v26h-4z" />
    <path d="m1006 271.703h46v26h-46z" />
    <path d="m1110 271.703h30v26h-30z" />
    <path d="m1057 271.703h30v26h-30z" />
    <path d="m1090 271.703h7v26h-7z" />
    <path d="m1100 271.703h8v26h-8z" />
    <path d="m964 271.703h3v26h-3z" />
    <path d="m979 271.703h21v26h-21z" />
    <path d="m1144 272h10v26h-10z" />
    <path d="m1 305.703h19v26h-19z" />
    <path d="m51 305.703h95v26h-95z" />
    <path d="m151 305.703h20v26h-20z" />
    <path d="m320 305.703h16v26h-16z" />
    <path d="m315 305.703h2v26h-2z" />
    <path d="m339 305.703h36v26h-36z" />
    <path d="m468 305.703h50v26h-50z" />
    <path d="m537 305.703h13v26h-13z" />
    <path d="m613 305.703h58v26h-58z" />
    <path d="m684 305.703h21v26h-21z" />
    <path d="m675 305.703h5v26h-5z" />
    <path d="m1006 305.703h46v26h-46z" />
    <path d="m1110 305.703h30v26h-30z" />
    <path d="m1057 305.703h30v26h-30z" />
    <path d="m1090 305.703h7v26h-7z" />
    <path d="m1100 305.703h8v26h-8z" />
    <path d="m964 305.703h3v26h-3z" />
    <path d="m979 305.703h21v26h-21z" />
    <path d="m863 305.703h4v26h-4z" />
    <path d="m1144 306h10v26h-10z" />
    <path d="m1 339.703h19v26h-19z" />
    <path d="m51 339.703h89v26h-89z" />
    <path d="m151 339.703h9v26h-9z" />
    <path d="m468 339.703h31v26h-31z" />
    <path d="m468 339.703h31v26h-31z" />
    <path d="m468 339.703h31v26h-31z" />
    <path d="m501 339.703h17v26h-17z" />
    <path d="m537 339.703h7v26h-7z" />
    <path d="m613 339.703h58v26h-58z" />
    <path d="m675 339.703h5v26h-5z" />
    <path d="m613 339.703h58v26h-58z" />
    <path d="m684 339.703h12v26h-12z" />
    <path d="m675 339.703h5v26h-5z" />
    <path d="m1006 339.703h46v26h-46z" />
    <path d="m1110 339.703h30v26h-30z" />
    <path d="m1057 339.703h30v26h-30z" />
    <path d="m1090 339.703h7v26h-7z" />
    <path d="m1100 339.703h8v26h-8z" />
    <path d="m964 339.703h3v26h-3z" />
    <path d="m979 339.703h21v26h-21z" />
    <path d="m863 339.703h4v26h-4z" />
    <path d="m1144 340h10v26h-10z" />
    <path d="m1 373.703h14v26h-14z" />
    <path d="m151 373.703h9v26h-9z" />
    <path d="m79 373.703h61v26h-61z" />
    <path d="m57 373.703h17v26h-17z" />
    <path d="m675 373.703h5v26h-5z" />
    <path d="m1008 373.703h27v26h-27z" />
    <path d="m964 373.703h3v26h-3z" />
    <path d="m979 373.703h21v26h-21z" />
    <path d="m863 373.703h4v26h-4z" />
    <path d="m1144 374h10v26h-10z" />
    <path d="m1 407.703h9v26h-9z" />
    <path d="m151 407.703h9v26h-9z" />
    <path d="m79 407.703h61v26h-61z" />
    <path d="m61 407.703h7v26h-7z" />
    <path d="m1010 407.703h23v26h-23z" />
    <path d="m964 407.703h3v26h-3z" />
    <path d="m979 407.703h21v26h-21z" />
    <path d="m1144 408h10v26h-10z" />
    <path d="m675 407.703h5v26h-5z" />
    <path d="m1 441.703h5v26h-5z" />
    <path d="m61 441.703h7v26h-7z" />
    <path d="m79 441.703h50v26h-50z" />
    <path d="m1012 441.703h17v26h-17z" />
    <path d="m964 441.703h3v26h-3z" />
    <path d="m979 441.703h21v26h-21z" />
    <path d="m1144 442h10v26h-10z" />
    <path d="m675 441.703h5v26h-5z" />
    <path d="m1 475.703h5v26h-5z" />
    <path d="m61 475.703h7v26h-7z" />
    <path d="m1015 475.703h11v26h-11z" />
    <path d="m964 475.703h3v26h-3z" />
    <path d="m979 475.703h21v26h-21z" />
    <path d="m1144 476h10v26h-10z" />
    <path d="m675 475.703h3v26h-3z" />
    <path d="m1 509.703h5v26h-5z" />
    <path d="m61 509.703h3v26h-3z" />
    <path d="m0 0h1v26h-1z" transform="matrix(-1 0 0 1 967 509.703)" />
    <path d="m979 509.703h21v26h-21z" />
    <path d="m1144 510h10v26h-10z" />
    <path d="m675 509.703h3v26h-3z" />
    <path d="m1 543.703h5v26h-5z" />
    <path d="m61 543.703h1v26h-1z" />
    <path d="m675 543.703h2v26h-2z" />
    <path d="m1144 544h10v26h-10z" />
    <path d="m1 68h45v26h-45z" />
    <path d="m51 68h259v26h-259z" />
    <path d="m315 68h146v26h-146z" />
    <path d="m468 68h370v26h-370z" />
    <path d="m843 68h15v26h-15z" />
    <path d="m863 68h85v26h-85z" />
    <path d="m954 67.7031h186v26h-186z" />
    <path d="m1144 68h10v26h-10z" />
    <path d="m1157 68h256v26h-256z" />
    <path d="m.5 34.293h460v25.7065h-460z" />
    <path d="m954 34h460v25.7065h-460z" />
    <path d="m468 34h480v25.7065h-480z" />
    <path d="m.5 0h1414v26h-1414z" />
  </g>
</svg>;

export const IS = () => (
  <svg
    fill="none"
    height="100%"
    viewBox="0 0 1455 688"
    width="100%"
    xmlns="http://www.w3.org/2000/svg"
  >
    <defs>
      <linearGradient id="shimmer" x1="0%" y1="0%" x2="100%" y2="0%">
        <stop offset="0.599964" stop-color="#f3f3f3" stop-opacity="1">
          <animate
            attributeName="offset"
            values="-2; -2; 1"
            keyTimes="0; 0.25; 1"
            dur="2s"
            repeatCount="indefinite"
          ></animate>
        </stop>
        <stop offset="1.59996" stop-color="#ecebeb" stop-opacity="1">
          <animate
            attributeName="offset"
            values="-1; -1; 2"
            keyTimes="0; 0.25; 1"
            dur="2s"
            repeatCount="indefinite"
          ></animate>
        </stop>
        <stop offset="2.59996" stop-color="#f3f3f3" stop-opacity="1">
          <animate
            attributeName="offset"
            values="0; 0; 3"
            keyTimes="0; 0.25; 1"
            dur="2s"
            repeatCount="indefinite"
          ></animate>
        </stop>
      </linearGradient>
    </defs>

    <g fill="url(#shimmer)">
      <path d="m1 101.703h19v26h-19z" />
      <path d="m51 101.703h254v26h-254z" />
      <path d="m315 101.703h99v26h-99z" />
      <path d="m420 101.703h41v26h-41z" />
      <path d="m468 101.703h260v26h-260z" />
      <path d="m734 101.703h104v26h-104z" />
      <path d="m843 101.703h11v26h-11z" />
      <path d="m863 101.703h73v26h-73z" />
      <path d="m1077 101.703h63v26h-63z" />
      <path d="m1007 101.703h70v26h-70z" />
      <path d="m954 102h46v26h-46z" />
      <path d="m1144 102h10v26h-10z" />
      <path d="m1159 101.703h24v26h-24z" />
      <path d="m1189 101.703h80v26h-80z" />
      <path d="m1274 101.703h139v26h-139z" />
      <path d="m1 135.703h19v26h-19z" />
      <path d="m51 135.703h218v26h-218z" />
      <path d="m274 135.703h25v26h-25z" />
      <path d="m315 135.703h85v26h-85z" />
      <path d="m441 135.703h8v26h-8z" />
      <path d="m453 135.703h8v26h-8z" />
      <path d="m405 135.703h8v26h-8z" />
      <path d="m417 135.703h8v26h-8z" />
      <path d="m429 135.703h8v26h-8z" />
      <path d="m468 135.703h138v26h-138z" />
      <path d="m613 135.703h115v26h-115z" />
      <path d="m734 135.703h61v26h-61z" />
      <path d="m800 135.703h38v26h-38z" />
      <path d="m843 135.703h7v26h-7z" />
      <path d="m863 135.703h28v26h-28z" />
      <path d="m899 135.703h33v26h-33z" />
      <path d="m1006 136h134v26h-134z" />
      <path d="m954 135.703h16v26h-16z" />
      <path d="m973 135.703h27v26h-27z" />
      <path d="m1144 136h10v26h-10z" />
      <path d="m1157 135.703h26v26h-26z" />
      <path d="m1189 135.703h80v26h-80z" />
      <path d="m1274 135.703h139v26h-139z" />
      <path d="m1 169.703h19v26h-19z" />
      <path d="m51 169.703h50v26h-50z" />
      <path d="m111 169.703h143v26h-143z" />
      <path d="m260 169.703h35v26h-35z" />
      <path d="m320 169.703h54v26h-54z" />
      <path d="m315 169.703h2v26h-2z" />
      <path d="m468 169.703h108v26h-108z" />
      <path d="m580 169.703h20v26h-20z" />
      <path d="m613 169.703h58v26h-58z" />
      <path d="m684 169.703h44v26h-44z" />
      <path d="m737 169.703h31v26h-31z" />
      <path d="m771 169.703h22v26h-22z" />
      <path d="m800 169.703h23v26h-23z" />
      <path d="m827 169.703h6v26h-6z" />
      <path d="m675 169.703h5v26h-5z" />
      <path d="m843 169.703h7v26h-7z" />
      <path d="m863 169.703h22v26h-22z" />
      <path d="m1006 169.703h46v26h-46z" />
      <path d="m1057 169.703h83v26h-83z" />
      <path d="m957 169.703h10v26h-10z" />
      <path d="m973 169.703h27v26h-27z" />
      <path d="m1144 170h10v26h-10z" />
      <path d="m1157 169.703h7v26h-7z" />
      <path d="m1168 169.703h7v26h-7z" />
      <path d="m1193 169.703h54v26h-54z" />
      <path d="m1274 169.703h54v26h-54z" />
      <path d="m1333 169.703h54v26h-54z" />
      <path d="m1392 169.703h7v26h-7z" />
      <path d="m1403 169.703h7v26h-7z" />
      <path d="m1256 169.703h13v26h-13z" />
      <path d="m1 203.703h19v26h-19z" />
      <path d="m51 203.703h141v26h-141z" />
      <path d="m198 203.703h94v26h-94z" />
      <path d="m320 203.703h55v26h-55z" />
      <path d="m315 203.703h2v26h-2z" />
      <path d="m468 203.703h64v26h-64z" />
      <path d="m613 203.703h58v26h-58z" />
      <path d="m684 203.703h40v26h-40z" />
      <path d="m675 203.703h5v26h-5z" />
      <path d="m537 203.703h39v26h-39z" />
      <path d="m581 203.703h9v26h-9z" />
      <path d="m843 203.703h5v26h-5z" />
      <path d="m863 203.703h15v26h-15z" />
      <path d="m1100 203.703h40v26h-40z" />
      <path d="m1057 203.703h40v26h-40z" />
      <path d="m1006 203.703h46v26h-46z" />
      <path d="m959 203.703h8v26h-8z" />
      <path d="m979 203.703h21v26h-21z" />
      <path d="m1144 204h10v26h-10z" />
      <path d="m1168 203.703h7v26h-7z" />
      <path d="m1 237.703h19v26h-19z" />
      <path d="m51 237.703h104v26h-104z" />
      <path d="m198 237.703h90v26h-90z" />
      <path d="m160 237.703h19v26h-19z" />
      <path d="m320 237.703h55v26h-55z" />
      <path d="m315 237.703h2v26h-2z" />
      <path d="m468 237.703h59v26h-59z" />
      <path d="m537 237.703h32v26h-32z" />
      <path d="m613 237.703h58v26h-58z" />
      <path d="m684 237.703h36v26h-36z" />
      <path d="m675 237.703h5v26h-5z" />
      <path d="m843 237.703h3v26h-3z" />
      <path d="m863 237.703h8v26h-8z" />
      <path d="m1006 237.703h46v26h-46z" />
      <path d="m1110 237.703h30v26h-30z" />
      <path d="m1057 237.703h30v26h-30z" />
      <path d="m1090 237.703h7v26h-7z" />
      <path d="m1090 237.703h7v26h-7z" />
      <path d="m1100 237.703h8v26h-8z" />
      <path d="m961 237.703h6v26h-6z" />
      <path d="m979 237.703h21v26h-21z" />
      <path d="m1144 238h10v26h-10z" />
      <path d="m1 271.703h19v26h-19z" />
      <path d="m51 271.703h128v26h-128z" />
      <path d="m198 271.703h84v26h-84z" />
      <path d="m320 271.703h55v26h-55z" />
      <path d="m315 271.703h2v26h-2z" />
      <path d="m468 271.703h50v26h-50z" />
      <path d="m537 271.703h21v26h-21z" />
      <path d="m613 271.703h58v26h-58z" />
      <path d="m684 271.703h29v26h-29z" />
      <path d="m675 271.703h5v26h-5z" />
      <path d="m843 271.703h1v26h-1z" />
      <path d="m863 271.703h4v26h-4z" />
      <path d="m1006 271.703h46v26h-46z" />
      <path d="m1110 271.703h30v26h-30z" />
      <path d="m1057 271.703h30v26h-30z" />
      <path d="m1090 271.703h7v26h-7z" />
      <path d="m1100 271.703h8v26h-8z" />
      <path d="m964 271.703h3v26h-3z" />
      <path d="m979 271.703h21v26h-21z" />
      <path d="m1144 272h10v26h-10z" />
      <path d="m1 305.703h19v26h-19z" />
      <path d="m51 305.703h95v26h-95z" />
      <path d="m151 305.703h20v26h-20z" />
      <path d="m320 305.703h16v26h-16z" />
      <path d="m315 305.703h2v26h-2z" />
      <path d="m339 305.703h36v26h-36z" />
      <path d="m468 305.703h50v26h-50z" />
      <path d="m537 305.703h13v26h-13z" />
      <path d="m613 305.703h58v26h-58z" />
      <path d="m684 305.703h21v26h-21z" />
      <path d="m675 305.703h5v26h-5z" />
      <path d="m1006 305.703h46v26h-46z" />
      <path d="m1110 305.703h30v26h-30z" />
      <path d="m1057 305.703h30v26h-30z" />
      <path d="m1090 305.703h7v26h-7z" />
      <path d="m1100 305.703h8v26h-8z" />
      <path d="m964 305.703h3v26h-3z" />
      <path d="m979 305.703h21v26h-21z" />
      <path d="m863 305.703h4v26h-4z" />
      <path d="m1144 306h10v26h-10z" />
      <path d="m1 339.703h19v26h-19z" />
      <path d="m51 339.703h89v26h-89z" />
      <path d="m151 339.703h9v26h-9z" />
      <path d="m468 339.703h31v26h-31z" />
      <path d="m468 339.703h31v26h-31z" />
      <path d="m468 339.703h31v26h-31z" />
      <path d="m501 339.703h17v26h-17z" />
      <path d="m537 339.703h7v26h-7z" />
      <path d="m613 339.703h58v26h-58z" />
      <path d="m675 339.703h5v26h-5z" />
      <path d="m613 339.703h58v26h-58z" />
      <path d="m684 339.703h12v26h-12z" />
      <path d="m675 339.703h5v26h-5z" />
      <path d="m1006 339.703h46v26h-46z" />
      <path d="m1110 339.703h30v26h-30z" />
      <path d="m1057 339.703h30v26h-30z" />
      <path d="m1090 339.703h7v26h-7z" />
      <path d="m1100 339.703h8v26h-8z" />
      <path d="m964 339.703h3v26h-3z" />
      <path d="m979 339.703h21v26h-21z" />
      <path d="m863 339.703h4v26h-4z" />
      <path d="m1144 340h10v26h-10z" />
      <path d="m1 373.703h14v26h-14z" />
      <path d="m151 373.703h9v26h-9z" />
      <path d="m79 373.703h61v26h-61z" />
      <path d="m57 373.703h17v26h-17z" />
      <path d="m675 373.703h5v26h-5z" />
      <path d="m1008 373.703h27v26h-27z" />
      <path d="m964 373.703h3v26h-3z" />
      <path d="m979 373.703h21v26h-21z" />
      <path d="m863 373.703h4v26h-4z" />
      <path d="m1144 374h10v26h-10z" />
      <path d="m1 407.703h9v26h-9z" />
      <path d="m151 407.703h9v26h-9z" />
      <path d="m79 407.703h61v26h-61z" />
      <path d="m61 407.703h7v26h-7z" />
      <path d="m1010 407.703h23v26h-23z" />
      <path d="m964 407.703h3v26h-3z" />
      <path d="m979 407.703h21v26h-21z" />
      <path d="m1144 408h10v26h-10z" />
      <path d="m675 407.703h5v26h-5z" />
      <path d="m1 441.703h5v26h-5z" />
      <path d="m61 441.703h7v26h-7z" />
      <path d="m79 441.703h50v26h-50z" />
      <path d="m1012 441.703h17v26h-17z" />
      <path d="m964 441.703h3v26h-3z" />
      <path d="m979 441.703h21v26h-21z" />
      <path d="m1144 442h10v26h-10z" />
      <path d="m675 441.703h5v26h-5z" />
      <path d="m1 475.703h5v26h-5z" />
      <path d="m61 475.703h7v26h-7z" />
      <path d="m1015 475.703h11v26h-11z" />
      <path d="m964 475.703h3v26h-3z" />
      <path d="m979 475.703h21v26h-21z" />
      <path d="m1144 476h10v26h-10z" />
      <path d="m675 475.703h3v26h-3z" />
      <path d="m1 509.703h5v26h-5z" />
      <path d="m61 509.703h3v26h-3z" />
      <path d="m0 0h1v26h-1z" transform="matrix(-1 0 0 1 967 509.703)" />
      <path d="m979 509.703h21v26h-21z" />
      <path d="m1144 510h10v26h-10z" />
      <path d="m675 509.703h3v26h-3z" />
      <path d="m1 543.703h5v26h-5z" />
      <path d="m61 543.703h1v26h-1z" />
      <path d="m675 543.703h2v26h-2z" />
      <path d="m1144 544h10v26h-10z" />
      <path d="m1 68h45v26h-45z" />
      <path d="m51 68h259v26h-259z" />
      <path d="m315 68h146v26h-146z" />
      <path d="m468 68h370v26h-370z" />
      <path d="m843 68h15v26h-15z" />
      <path d="m863 68h85v26h-85z" />
      <path d="m954 67.7031h186v26h-186z" />
      <path d="m1144 68h10v26h-10z" />
      <path d="m1157 68h256v26h-256z" />
      <path d="m.5 34.293h460v25.7065h-460z" />
      <path d="m954 34h460v25.7065h-460z" />
      <path d="m468 34h480v25.7065h-480z" />
      <path d="m.5 0h1414v26h-1414z" />
    </g>
  </svg>
);

export const TS = () => (
  <svg
    fill="none"
    height="100%"
    viewBox="0 0 1415 605"
    width="100%"
    xmlns="http://www.w3.org/2000/svg"
  >
    <defs>
      <linearGradient id="table-data" x1="0%" y1="0%" x2="100%" y2="0%">
        <stop offset="0.599964" stop-color="#f3f3f3" stop-opacity="1">
          <animate
            attributeName="offset"
            values="-2; -2; 1"
            keyTimes="0; 0.25; 1"
            dur="2s"
            repeatCount="indefinite"
          ></animate>
        </stop>
        <stop offset="1.59996" stop-color="#ecebeb" stop-opacity="1">
          <animate
            attributeName="offset"
            values="-1; -1; 2"
            keyTimes="0; 0.25; 1"
            dur="2s"
            repeatCount="indefinite"
          ></animate>
        </stop>
        <stop offset="2.59996" stop-color="#f3f3f3" stop-opacity="1">
          <animate
            attributeName="offset"
            values="0; 0; 3"
            keyTimes="0; 0.25; 1"
            dur="2s"
            repeatCount="indefinite"
          ></animate>
        </stop>
      </linearGradient>
    </defs>
    <rect fill="url(#table-data)" id="table-data" height="8" rx="4" width="39" x="18" y="46.5" />
    <rect fill="url(#table-data)" id="table-data" height="8" rx="4" width="71" x="110" y="46.5" />
    <path d="m1 64.5 1412 .0003" stroke="#e5e7eb" />
    <g fill="url(#table-data)" id="table-data">
      <rect height="8" rx="4" width="400" x="268" y="46.5" />
      <rect height="8" rx="4" width="39" x="18" y="76.5" />
      <rect height="8" rx="4" width="71" x="110" y="76.5" />
    </g>
    <path d="m1 94.5 1412 .0003" stroke="#e5e7eb" />
    <rect fill="url(#table-data)" id="table-data" height="8" rx="4" width="400" x="268" y="76.5" />
    <rect fill="url(#table-data)" id="table-data" height="8" rx="4" width="39" x="18" y="106.5" />
    <rect fill="url(#table-data)" id="table-data" height="8" rx="4" width="71" x="110" y="106.5" />
    <path d="m1 124.5h1412" stroke="#e5e7eb" />
    <rect fill="url(#table-data)" id="table-data" height="8" rx="4" width="400" x="268" y="106.5" />
    <rect fill="url(#table-data)" id="table-data" height="8" rx="4" width="39" x="18" y="136.5" />
    <rect fill="url(#table-data)" id="table-data" height="8" rx="4" width="71" x="110" y="136.5" />
    <path d="m1 154.5h1412" stroke="#e5e7eb" />
    <rect fill="url(#table-data)" id="table-data" height="8" rx="4" width="400" x="268" y="136.5" />
    <rect fill="url(#table-data)" id="table-data" height="8" rx="4" width="39" x="18" y="166.5" />
    <rect fill="url(#table-data)" id="table-data" height="8" rx="4" width="71" x="110" y="166.5" />
    <path d="m1 184.5h1412" stroke="#e5e7eb" />
    <rect fill="url(#table-data)" id="table-data" height="8" rx="4" width="400" x="268" y="166.5" />
    <rect fill="url(#table-data)" id="table-data" height="8" rx="4" width="39" x="18" y="196.5" />
    <rect fill="url(#table-data)" id="table-data" height="8" rx="4" width="71" x="110" y="196.5" />
    <path d="m1 214.5h1412" stroke="#e5e7eb" />
    <rect fill="url(#table-data)" id="table-data" height="8" rx="4" width="400" x="268" y="196.5" />
    <rect fill="url(#table-data)" id="table-data" height="8" rx="4" width="39" x="18" y="226.5" />
    <rect fill="url(#table-data)" id="table-data" height="8" rx="4" width="71" x="110" y="226.5" />
    <path d="m1 244.5h1412" stroke="#e5e7eb" />
    <rect fill="url(#table-data)" id="table-data" height="8" rx="4" width="400" x="268" y="226.5" />
    <rect fill="url(#table-data)" id="table-data" height="8" rx="4" width="39" x="18" y="256.5" />
    <rect fill="url(#table-data)" id="table-data" height="8" rx="4" width="71" x="110" y="256.5" />
    <path d="m1 274.5h1412" stroke="#e5e7eb" />
    <rect fill="url(#table-data)" id="table-data" height="8" rx="4" width="400" x="268" y="256.5" />
    <rect fill="url(#table-data)" id="table-data" height="8" rx="4" width="39" x="18" y="286.5" />
    <rect fill="url(#table-data)" id="table-data" height="8" rx="4" width="71" x="110" y="286.5" />
    <path d="m1 304.5h1412" stroke="#e5e7eb" />
    <rect fill="url(#table-data)" id="table-data" height="8" rx="4" width="400" x="268" y="286.5" />
    <rect fill="url(#table-data)" id="table-data" height="8" rx="4" width="39" x="18" y="316.5" />
    <rect fill="url(#table-data)" id="table-data" height="8" rx="4" width="71" x="110" y="316.5" />
    <path d="m1 334.5h1412" stroke="#e5e7eb" />
    <rect fill="url(#table-data)" id="table-data" height="8" rx="4" width="400" x="268" y="316.5" />
    <rect fill="url(#table-data)" id="table-data" height="8" rx="4" width="39" x="18" y="346.5" />
    <rect fill="url(#table-data)" id="table-data" height="8" rx="4" width="71" x="110" y="346.5" />
    <path d="m1 364.5h1412" stroke="#e5e7eb" />
    <rect fill="url(#table-data)" id="table-data" height="8" rx="4" width="400" x="268" y="346.5" />
    <rect fill="url(#table-data)" id="table-data" height="8" rx="4" width="39" x="18" y="376.5" />
    <rect fill="url(#table-data)" id="table-data" height="8" rx="4" width="71" x="110" y="376.5" />
    <path d="m1 394.5h1412" stroke="#e5e7eb" />
    <rect fill="url(#table-data)" id="table-data" height="8" rx="4" width="400" x="268" y="376.5" />
    <rect fill="url(#table-data)" id="table-data" height="8" rx="4" width="39" x="18" y="406.5" />
    <rect fill="url(#table-data)" id="table-data" height="8" rx="4" width="71" x="110" y="406.5" />
    <path d="m1 424.5h1412" stroke="#e5e7eb" />
    <rect fill="url(#table-data)" id="table-data" height="8" rx="4" width="400" x="268" y="406.5" />
    <rect fill="url(#table-data)" id="table-data" height="8" rx="4" width="39" x="18" y="436.5" />
    <rect fill="url(#table-data)" id="table-data" height="8" rx="4" width="71" x="110" y="436.5" />
    <path d="m1 454.5h1412" stroke="#e5e7eb" />
    <rect fill="url(#table-data)" id="table-data" height="8" rx="4" width="400" x="268" y="436.5" />
    <rect fill="url(#table-data)" id="table-data" height="8" rx="4" width="39" x="18" y="466.5" />
    <rect fill="url(#table-data)" id="table-data" height="8" rx="4" width="71" x="110" y="466.5" />
    <path d="m1 484.5h1412" stroke="#e5e7eb" />
    <rect fill="url(#table-data)" id="table-data" height="8" rx="4" width="400" x="268" y="466.5" />
    <rect fill="url(#table-data)" id="table-data" height="8" rx="4" width="39" x="18" y="496.5" />
    <rect fill="url(#table-data)" id="table-data" height="8" rx="4" width="71" x="110" y="496.5" />
    <path d="m1 514.5h1412" stroke="#e5e7eb" />
    <rect fill="url(#table-data)" id="table-data" height="8" rx="4" width="400" x="268" y="496.5" />
    <rect fill="url(#table-data)" id="table-data" height="8" rx="4" width="39" x="18" y="526.5" />
    <rect fill="url(#table-data)" id="table-data" height="8" rx="4" width="71" x="110" y="526.5" />
    <path d="m1 544.5h1412" stroke="#e5e7eb" />
    <rect fill="url(#table-data)" id="table-data" height="8" rx="4" width="400" x="268" y="526.5" />
    <rect fill="url(#table-data)" id="table-data" height="8" rx="4" width="39" x="18" y="556.5" />
    <rect fill="url(#table-data)" id="table-data" height="8" rx="4" width="71" x="110" y="556.5" />
    <path d="m1 574.5h1412" stroke="#e5e7eb" />
    <rect fill="url(#table-data)" id="table-data" height="8" rx="4" width="400" x="268" y="556.5" />
    <rect fill="url(#table-data)" id="table-data" height="8" rx="4" width="39" x="18" y="586.5" />
    <rect fill="url(#table-data)" id="table-data" height="8" rx="4" width="71" x="110" y="586.5" />
    <path d="m1 604.5h1412" stroke="#e5e7eb" />
    <rect fill="url(#table-data)" id="table-data" height="8" rx="4" width="400" x="268" y="586.5" />
    <path d="m.5 0h1414v36h-1414z" fill="#ebebeb" />
    <path
      d="m21.0043 23v-10.1818h6.5227v1.5461h-4.6783v2.7643h4.2308v1.5461h-4.2308v4.3253zm9.9233-10.1818v10.1818h-1.7998v-10.1818zm4.0481 10.3359c-.4839 0-.9198-.0862-1.3076-.2585-.3844-.1757-.6893-.4342-.9147-.7756-.2221-.3414-.3331-.7623-.3331-1.2628 0-.4308.0795-.7871.2386-1.0688.1591-.2818.3762-.5072.6513-.6762s.585-.2966.9297-.3828c.348-.0895.7076-.1541 1.0788-.1939.4475-.0464.8104-.0878 1.0888-.1243.2784-.0398.4806-.0994.6065-.179.1293-.0828.1939-.2104.1939-.3828v-.0298c0-.3745-.111-.6645-.3331-.87-.222-.2055-.5419-.3083-.9595-.3083-.4408 0-.7905.0962-1.049.2884-.2552.1922-.4276.4193-.517.6811l-1.6804-.2386c.1325-.4641.3513-.8518.6562-1.1634.3049-.3149.6778-.5502 1.1186-.706.4408-.159.928-.2386 1.4617-.2386.3679 0 .7341.0431 1.0987.1293.3646.0861.6977.2287.9993.4275.3016.1956.5435.4624.7258.8004.1856.3381.2784.7607.2784 1.2678v5.1108h-1.7301v-1.049h-.0596c-.1094.2121-.2635.411-.4624.5966-.1955.1823-.4425.3298-.7408.4425-.2949.1093-.6413.164-1.039.164zm.4673-1.3224c.3613 0 .6745-.0713.9396-.2138.2652-.1458.469-.3381.6115-.5767.1459-.2386.2188-.4988.2188-.7805v-.8999c-.0564.0464-.1525.0895-.2884.1293-.1325.0397-.2817.0745-.4474.1044-.1657.0298-.3298.0563-.4922.0795s-.3033.0431-.4226.0597c-.2684.0364-.5087.0961-.7209.1789-.2121.0829-.3795.1989-.5021.3481-.1226.1458-.1839.3347-.1839.5667 0 .3315.1209.5817.3629.7507.2419.1691.5502.2536.9247.2536zm9.1875-6.4681v1.3921h-4.3899v-1.3921zm-3.3061-1.8295h1.7997v7.169c0 .242.0365.4276.1094.5568.0762.126.1757.2122.2983.2586s.2585.0696.4077.0696c.1126 0 .2154-.0083.3082-.0249.0961-.0166.169-.0315.2187-.0447l.3033 1.4069c-.0961.0332-.2337.0696-.4126.1094-.1757.0398-.3911.063-.6463.0696-.4508.0133-.8568-.0547-1.2181-.2038-.3612-.1525-.6479-.3878-.8601-.706-.2088-.3182-.3115-.7159-.3082-1.1932z"
      fill="#000"
    />
    <path
      d="m118.719 16.2536h-1.859c-.053-.305-.151-.5751-.293-.8104-.143-.2387-.32-.4408-.532-.6066-.212-.1657-.454-.29-.726-.3728-.269-.0862-.559-.1293-.87-.1293-.554 0-1.044.1392-1.472.4176-.427.2751-.762.6795-1.004 1.2131-.242.5303-.363 1.1783-.363 1.9439 0 .7789.121 1.4351.363 1.9687.245.5303.58.9314 1.004 1.2032.428.2684.917.4027 1.467.4027.305 0 .59-.0398.855-.1194.269-.0828.509-.2038.721-.3629.215-.1591.396-.3546.542-.5866.149-.232.252-.4972.308-.7955l1.859.01c-.069.4839-.22.9379-.452 1.3622-.229.4242-.529.7987-.9 1.1236-.371.3215-.805.5733-1.302.7556-.498.179-1.049.2685-1.656.2685-.895 0-1.694-.2071-2.396-.6214-.703-.4143-1.256-1.0126-1.661-1.7948-.404-.7822-.606-1.7202-.606-2.8139 0-1.0971.204-2.035.611-2.8139.408-.7822.963-1.3805 1.666-1.7948.702-.4143 1.498-.6214 2.386-.6214.567 0 1.094.0795 1.581.2386s.921.3928 1.303.701c.381.3049.694.6795.939 1.1236.249.4408.411.9446.487 1.5114zm6.477 3.5348v-4.4248h1.8v7.6364h-1.745v-1.3572h-.08c-.172.4275-.455.7772-.85 1.049-.391.2717-.873.4076-1.447.4076-.5 0-.942-.111-1.327-.3331-.381-.2253-.679-.5518-.895-.9794-.215-.4308-.323-.9512-.323-1.561v-4.8623h1.8v4.5838c0 .4839.132.8684.397 1.1535.266.285.614.4275 1.044.4275.266 0 .523-.0646.771-.1939.249-.1292.453-.3215.612-.5767.162-.2585.243-.5817.243-.9694zm3.651 3.2116v-7.6364h1.72v1.2976h.089c.159-.4375.423-.7789.791-1.0241.368-.2486.807-.3729 1.317-.3729.517 0 .953.126 1.308.3778.358.2486.61.5884.755 1.0192h.08c.169-.4242.454-.7623.855-1.0142.404-.2552.883-.3828 1.437-.3828.703 0 1.276.2221 1.72.6662s.666 1.0921.666 1.9439v5.1257h-1.804v-4.8473c0-.474-.126-.8203-.378-1.0391-.252-.222-.56-.3331-.925-.3331-.434 0-.774.1359-1.019.4077-.242.2685-.363.6181-.363 1.049v4.7628h-1.765v-4.9219c0-.3944-.119-.7093-.358-.9446-.235-.2353-.544-.353-.925-.353-.258 0-.494.0663-.706.1989-.212.1293-.381.3132-.507.5518-.126.2354-.189.5105-.189.8253v4.6435zm17.431-3.2116v-4.4248h1.8v7.6364h-1.745v-1.3572h-.08c-.172.4275-.455.7772-.85 1.049-.391.2717-.873.4076-1.447.4076-.5 0-.942-.111-1.327-.3331-.381-.2253-.679-.5518-.895-.9794-.215-.4308-.323-.9512-.323-1.561v-4.8623h1.8v4.5838c0 .4839.132.8684.397 1.1535.266.285.614.4275 1.045.4275.265 0 .522-.0646.77-.1939.249-.1292.453-.3215.612-.5767.162-.2585.243-.5817.243-.9694zm5.45-6.9702v10.1818h-1.799v-10.1818zm4.048 10.3359c-.483 0-.919-.0862-1.307-.2585-.385-.1757-.689-.4342-.915-.7756-.222-.3414-.333-.7623-.333-1.2628 0-.4308.08-.7871.239-1.0688.159-.2818.376-.5072.651-.6762s.585-.2966.93-.3828c.348-.0895.707-.1541 1.078-.1939.448-.0464.811-.0878 1.089-.1243.279-.0398.481-.0994.607-.179.129-.0828.194-.2104.194-.3828v-.0298c0-.3745-.111-.6645-.333-.87s-.542-.3083-.96-.3083c-.441 0-.79.0962-1.049.2884-.255.1922-.427.4193-.517.6811l-1.68-.2386c.132-.4641.351-.8518.656-1.1634.305-.3149.678-.5502 1.118-.706.441-.159.929-.2386 1.462-.2386.368 0 .734.0431 1.099.1293.364.0861.698.2287.999.4275.302.1956.544.4624.726.8004.186.3381.278.7607.278 1.2678v5.1108h-1.73v-1.049h-.059c-.11.2121-.264.411-.463.5966-.195.1823-.442.3298-.74.4425-.295.1093-.642.164-1.04.164zm.468-1.3224c.361 0 .674-.0713.939-.2138.266-.1458.469-.3381.612-.5767.146-.2386.219-.4988.219-.7805v-.8999c-.057.0464-.153.0895-.289.1293-.132.0397-.281.0745-.447.1044-.166.0298-.33.0563-.492.0795-.163.0232-.304.0431-.423.0597-.268.0364-.509.0961-.721.1789-.212.0829-.379.1989-.502.3481-.123.1458-.184.3347-.184.5667 0 .3315.121.5817.363.7507.242.1691.55.2536.925.2536zm9.187-6.4681v1.3921h-4.39v-1.3921zm-3.306-1.8295h1.8v7.169c0 .242.036.4276.109.5568.076.126.176.2122.299.2586.122.0464.258.0696.407.0696.113 0 .216-.0083.308-.0249.097-.0166.17-.0315.219-.0447l.303 1.4069c-.096.0332-.233.0696-.412.1094-.176.0398-.391.063-.646.0696-.451.0133-.857-.0547-1.218-.2038-.362-.1525-.648-.3878-.861-.706-.208-.3182-.311-.7159-.308-1.1932zm4.811 9.4659v-7.6364h1.8v7.6364zm.905-8.7202c-.285 0-.53-.0944-.736-.2834-.205-.1922-.308-.4225-.308-.691 0-.2718.103-.5021.308-.6911.206-.1922.451-.2883.736-.2883.289 0 .534.0961.736.2883.206.189.308.4193.308.6911 0 .2685-.102.4988-.308.691-.202.189-.447.2834-.736.2834zm9.567 1.0838-2.72 7.6364h-1.988l-2.72-7.6364h1.919l1.755 5.6726h.08l1.76-5.6726zm4.366 7.7855c-.765 0-1.427-.159-1.984-.4772-.553-.3215-.979-.7756-1.277-1.3622-.299-.59-.448-1.2844-.448-2.0831 0-.7855.149-1.4749.448-2.0682.301-.5966.722-1.0606 1.263-1.3921.54-.3347 1.175-.5021 1.904-.5021.47 0 .914.0762 1.332.2287.421.1491.792.3812 1.114.696.325.3149.58.7159.765 1.2031.186.4839.279 1.0607.279 1.7302v.5518h-6.259v-1.2131h4.534c-.004-.3447-.078-.6512-.224-.9197-.146-.2718-.35-.4856-.612-.6413-.258-.1558-.56-.2337-.904-.2337-.368 0-.692.0895-.97.2685-.278.1756-.495.4076-.651.696-.153.285-.231.5982-.234.9396v1.059c0 .4441.081.8252.244 1.1434.162.3149.389.5568.681.7259.292.1657.633.2486 1.024.2486.262 0 .499-.0365.711-.1094.212-.0762.396-.1873.552-.3331s.273-.3265.353-.5419l1.68.1889c-.106.4441-.308.8319-.606 1.1634-.295.3281-.673.5833-1.134.7656-.461.179-.988.2684-1.581.2684z"
      fill="#000"
    />
    <path
      d="m278.371 12.8182v10.1818h-1.64l-4.798-6.9354h-.084v6.9354h-1.845v-10.1818h1.651l4.792 6.9403h.09v-6.9403zm4.128 10.3359c-.484 0-.92-.0862-1.307-.2585-.385-.1757-.69-.4342-.915-.7756-.222-.3414-.333-.7623-.333-1.2628 0-.4308.079-.7871.238-1.0688.159-.2818.377-.5072.652-.6762s.585-.2966.929-.3828c.348-.0895.708-.1541 1.079-.1939.448-.0464.811-.0878 1.089-.1243.278-.0398.481-.0994.606-.179.13-.0828.194-.2104.194-.3828v-.0298c0-.3745-.111-.6645-.333-.87s-.542-.3083-.959-.3083c-.441 0-.791.0962-1.049.2884-.255.1922-.428.4193-.517.6811l-1.681-.2386c.133-.4641.352-.8518.657-1.1634.304-.3149.677-.5502 1.118-.706.441-.159.928-.2386 1.462-.2386.368 0 .734.0431 1.099.1293.364.0861.697.2287.999.4275.301.1956.543.4624.726.8004.185.3381.278.7607.278 1.2678v5.1108h-1.73v-1.049h-.06c-.109.2121-.263.411-.462.5966-.196.1823-.443.3298-.741.4425-.295.1093-.641.164-1.039.164zm.467-1.3224c.362 0 .675-.0713.94-.2138.265-.1458.469-.3381.612-.5767.145-.2386.218-.4988.218-.7805v-.8999c-.056.0464-.152.0895-.288.1293-.133.0397-.282.0745-.447.1044-.166.0298-.33.0563-.493.0795-.162.0232-.303.0431-.422.0597-.269.0364-.509.0961-.721.1789-.212.0829-.38.1989-.502.3481-.123.1458-.184.3347-.184.5667 0 .3315.121.5817.363.7507.242.1691.55.2536.924.2536zm5.375 1.1683v-7.6364h1.72v1.2976h.089c.159-.4375.423-.7789.791-1.0241.368-.2486.807-.3729 1.317-.3729.517 0 .953.126 1.308.3778.358.2486.61.5884.756 1.0192h.079c.169-.4242.454-.7623.855-1.0142.405-.2552.884-.3828 1.437-.3828.703 0 1.276.2221 1.72.6662s.666 1.0921.666 1.9439v5.1257h-1.804v-4.8473c0-.474-.126-.8203-.378-1.0391-.252-.222-.56-.3331-.925-.3331-.434 0-.774.1359-1.019.4077-.242.2685-.363.6181-.363 1.049v4.7628h-1.765v-4.9219c0-.3944-.119-.7093-.358-.9446-.235-.2353-.543-.353-.925-.353-.258 0-.493.0663-.706.1989-.212.1293-.381.3132-.507.5518-.126.2354-.189.5105-.189.8253v4.6435zm15.945.1491c-.766 0-1.427-.159-1.984-.4772-.553-.3215-.979-.7756-1.277-1.3622-.299-.59-.448-1.2844-.448-2.0831 0-.7855.149-1.4749.448-2.0682.301-.5966.722-1.0606 1.262-1.3921.541-.3347 1.175-.5021 1.904-.5021.471 0 .915.0762 1.333.2287.421.1491.792.3812 1.113.696.325.3149.58.7159.766 1.2031.186.4839.278 1.0607.278 1.7302v.5518h-6.259v-1.2131h4.534c-.003-.3447-.078-.6512-.223-.9197-.146-.2718-.35-.4856-.612-.6413-.258-.1558-.56-.2337-.905-.2337-.368 0-.691.0895-.969.2685-.279.1756-.496.4076-.651.696-.153.285-.231.5982-.234.9396v1.059c0 .4441.081.8252.243 1.1434.163.3149.39.5568.682.7259.291.1657.633.2486 1.024.2486.262 0 .499-.0365.711-.1094.212-.0762.396-.1873.552-.3331.155-.1458.273-.3265.352-.5419l1.681.1889c-.106.4441-.308.8319-.607 1.1634-.295.3281-.672.5833-1.133.7656-.461.179-.988.2684-1.581.2684z"
      fill="#000"
    />
  </svg>
);

const Spinner = (): JSX.Element => {
  return (
    <div
      style={{
        display: 'flex',
        justifyContent: 'center',
        alignItems: 'center',
        height: 'inherit',
        paddingTop: 40,
        paddingBottom: 40,
      }}
    >
      <svg
        className="-ml-1 mr-3 h-5 w-5 animate-spin"
        xmlns="http://www.w3.org/2000/svg"
        fill="none"
        viewBox="0 0 24 24"
      >
        <circle
          className="opacity-25"
          cx="12"
          cy="12"
          r="10"
          stroke="currentColor"
          strokeWidth="4"
        ></circle>
        <path
          className="opacity-75"
          fill="currentColor"
          d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
        ></path>
      </svg>
      <span>Loading...</span>
    </div>
    // <IS />
  );
};

export default Spinner;
