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

import cx from 'classnames';

interface Props {
  isDarkMode: boolean;
  isHalfScreen?: boolean;
}

const SandwichFlameGraphSkeleton = ({isDarkMode, isHalfScreen}: Props): JSX.Element => {
  return (
    <svg
      fill="none"
      height="100%"
      viewBox="0 0 2000 509"
      width={isHalfScreen === true ? '1455px' : '100%'}
      xmlns="http://www.w3.org/2000/svg"
    >
      <defs>
        <linearGradient id="shimmer-flame" x1="0%" y1="0%" x2="100%" y2="0%">
          <stop
            offset="0.599964"
            stopColor={cx(isDarkMode ? '#1f2937' : '#f3f3f3')}
            stopOpacity="1"
          >
            <animate
              attributeName="offset"
              values="-2; -2; 1"
              keyTimes="0; 0.25; 1"
              dur="2s"
              repeatCount="indefinite"
            ></animate>
          </stop>
          <stop offset="1.59996" stopColor={cx(isDarkMode ? '#374151' : '#ecebeb')} stopOpacity="1">
            <animate
              attributeName="offset"
              values="-1; -1; 2"
              keyTimes="0; 0.25; 1"
              dur="2s"
              repeatCount="indefinite"
            ></animate>
          </stop>
          <stop offset="2.59996" stopColor={cx(isDarkMode ? '#1f2937' : '#f3f3f3')} stopOpacity="1">
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

      <g fill="url(#shimmer-flame)">
        {/* Root level (bottom) */}
        <path d="m0 483h2000v26h-2000z" />

        {/* Level 1 */}
        <path d="m0 452h662.667v26h-662.667z" />
        <path d="m668.667 452h662.667v26h-662.667z" />
        <path d="m1337.33 452h662.667v26h-662.667z" />

        {/* Level 2 */}
        <path d="m0 421h63.7394v26h-63.7394z" />
        <path d="m70.8218 421h366.856v26h-366.856z" />
        <path d="m445 421h216v26h-216z" />
        <path d="m671 421h515v26h-515z" />
        <path d="m1192.63 421h21.2465v26h-21.2465z" />
        <path d="m1220.41 421h90.7158v26h-90.7158z" />
        <path d="m1337 421h276v26h-276z" />
        <path d="m1618.98 421h14.1643v26h-14.1643z" />
        <path d="m1637.39 421h362.606v26h-362.606z" />

        {/* Level 3 */}
        <path d="m0 390h26.9122v26h-26.9122z" />
        <path d="m73 390h358v26h-358z" />
        <path d="m444.759 390h140.227v26h-140.227z" />
        <path d="m593.484 390h58.0737v26h-58.0737z" />
        <path d="m673 390h357v26h-357z" />
        <path d="m1038.24 390h147.309v26h-147.309z" />
        <path d="m1192.63 390h15.5807v26h-15.5807z" />
        <path d="m1220.41 390h90.7158v26h-90.7158z" />
        <path d="m1524.08 390h89.2352v26h-89.2352z" />
        <path d="m1424.93 390h99.1502v26h-99.1502z" />
        <path d="m1337 390h78v26h-78z" />
        <path d="m1618.98 390h14.1643v26h-14.1643z" />
        <path d="m1640.23 390h33.9943v26h-33.9943z" />
        <path d="m1682.72 390h113.314v26h-113.314z" />
        <path d="m1803.12 390h196.884v26h-196.884z" />

        {/* Level 4 */}
        <path d="m0 359h26.9122v26h-26.9122z" />
        <path d="m73 359h307v26h-307z" />
        <path d="m386.686 359h35.4108v26h-35.4108z" />
        <path d="m444.759 359h120.397v26h-120.397z" />
        <path d="m623.229 359h11.3314v26h-11.3314z" />
        <path d="m640.227 359h11.3314v26h-11.3314z" />
        <path d="m593 359h8v26h-8z" />
        <path d="m606.232 359h11.3314v26h-11.3314z" />
        <path d="m673 359h184v26h-184z" />
        <path d="m866.856 359h162.89v26h-162.89z" />
        <path d="m1038.24 359h86.4023v26h-86.4023z" />
        <path d="m1131.73 359h53.8244v26h-53.8244z" />
        <path d="m1192.63 359h9.91502v26h-9.91502z" />
        <path d="m1220.96 359h39.6601v26h-39.6601z" />
        <path d="m1271.44 359h39.6882v26h-39.6882z" />
        <path d="m1423.51 359h189.802v26h-189.802z" />
        <path d="m1349.86 359h22.6629v26h-22.6629z" />
        <path d="m1376.77 359h38.2436v26h-38.2436z" />
        <path d="m1618.98 359h14.1643v26h-14.1643z" />
        <path d="m1637.39 359h36.8272v26h-36.8272z" />
        <path d="m1682.72 359h113.314v26h-113.314z" />
        <path d="m1803.12 359h196.884v26h-196.884z" />

        {/* Level 5 */}
        <path d="m0 328h26.9695v26h-26.9695z" />
        <path d="m73 328h69v26h-69z" />
        <path d="m156.139 328h202.981v26h-202.981z" />
        <path d="m387 328h30v26h-30z" />
        <path d="m453 328h112v26h-112z" />
        <path d="m445.706 328h2.83889v26h-2.83889z" />
        <path d="m673 328h143v26h-143z" />
        <path d="m821.859 328h28.3889v26h-28.3889z" />
        <path d="m868.701 328h82.3279v26h-82.3279z" />
        <path d="m969.481 328h62.4556v26h-62.4556z" />
        <path d="m1044.71 328h44.0028v26h-44.0028z" />
        <path d="m1092.97 328h31.2278v26h-31.2278z" />
        <path d="m1134.14 328h32.6473v26h-32.6473z" />
        <path d="m1172.46 328h8.51668v26h-8.51668z" />
        <path d="m956.707 328h7.09723v26h-7.09723z" />
        <path d="m1192.17 328h9.93613v26h-9.93613z" />
        <path d="m1223.56 328h31.2278v26h-31.2278z" />
        <path d="m1424.54 328h65.2945v26h-65.2945z" />
        <path d="m1499 328h114v26h-114z" />
        <path d="m1357.99 328h14.1945v26h-14.1945z" />
        <path d="m1376.7 328h38.3251v26h-38.3251z" />
        <path d="m1619 328h14v26h-14z" />
        <path d="m1640.88 328h9.93613v26h-9.93613z" />
        <path d="m1656.49 328h9.93613v26h-9.93613z" />
        <path d="m1691.98 328h76.6501v26h-76.6501z" />
        <path d="m1806.96 328h76.6501v26h-76.6501z" />
        <path d="m1890.7 328h76.6501v26h-76.6501z" />
        <path d="m1974.45 328h9.93613v26h-9.93613z" />
        <path d="m1990.06 328h9.93613v26h-9.93613z" />
        <path d="m1781.41 328h18.4528v26h-18.4528z" />

        {/* Level 6 */}
        <path d="m0 297h19.4693v26h-19.4693z" />
        <path d="m73 297h69v26h-69z" />
        <path d="m156.866 297h96.322v26h-96.322z" />
        <path d="m262.88 297h56.3586v26h-56.3586z" />
        <path d="m387 297h30v26h-30z" />
        <path d="m257.756 297h2.0494v26h-2.0494z" />
        <path d="m453.536 297h65.5809v26h-65.5809z" />
        <path d="m678 297h58v26h-58z" />
        <path d="m748.871 297h40.9881v26h-40.9881z" />
        <path d="m740 297h5v26h-5z" />
        <path d="m522.24 297h39.9634v26h-39.9634z" />

        {/* Level 7 */}
        <path d="m0 266h15v26h-15z" />
        <path d="m73 266h45v26h-45z" />
        <path d="m156 266h80v26h-80z" />
        <path d="m453 266h50v26h-50z" />
        <path d="m678 266h40v26h-40z" />
        <path d="m1170 266h60v26h-60z" />
        <path d="m1500 266h90v26h-90z" />

        {/* Level 8 */}
        <path d="m0 235h12v26h-12z" />
        <path d="m73 235h35v26h-35z" />
        <path d="m156 235h60v26h-60z" />
        <path d="m453 235h35v26h-35z" />
        <path d="m678 235h25v26h-25z" />
        <path d="m1170 235h40v26h-40z" />

        {/* Level 9 */}
        <path d="m0 204h10v26h-10z" />
        <path d="m73 204h25v26h-25z" />
        <path d="m156 204h40v26h-40z" />
        <path d="m453 204h20v26h-20z" />
        <path d="m678 204h15v26h-15z" />
        <path d="m1170 204h25v26h-25z" />

        {/* Level 10 */}
        <path d="m0 173h8v26h-8z" />
        <path d="m73 173h20v26h-20z" />
        <path d="m156 173h25v26h-25z" />
        <path d="m453 173h15v26h-15z" />
        <path d="m1170 173h20v26h-20z" />

        {/* Level 11 */}
        <path d="m0 142h6v26h-6z" />
        <path d="m73 142h15v26h-15z" />
        <path d="m156 142h20v26h-20z" />
        <path d="m453 142h12v26h-12z" />
        <path d="m1170 142h15v26h-15z" />

        {/* Level 12 */}
        <path d="m0 111h5v26h-5z" />
        <path d="m73 111h12v26h-12z" />
        <path d="m156 111h15v26h-15z" />
        <path d="m453 111h10v26h-10z" />
        <path d="m1170 111h12v26h-12z" />

        {/* Level 13 */}
        <path d="m0 80h4v26h-4z" />
        <path d="m73 80h10v26h-10z" />
        <path d="m156 80h12v26h-12z" />
        <path d="m453 80h8v26h-8z" />
        <path d="m1170 80h10v26h-10z" />

        {/* Level 14 */}
        <path d="m0 49h3v26h-3z" />
        <path d="m73 49h8v26h-8z" />
        <path d="m156 49h10v26h-10z" />
        <path d="m453 49h6v26h-6z" />
        <path d="m1170 49h8v26h-8z" />

        {/* Level 15 */}
        <path d="m0 18h2v26h-2z" />
        <path d="m73 18h5v26h-5z" />
        <path d="m156 18h6v26h-6z" />
        <path d="m453 18h4v26h-4z" />
        <path d="m1170 18h5v26h-5z" />
      </g>
    </svg>
  );
};

export default SandwichFlameGraphSkeleton;
