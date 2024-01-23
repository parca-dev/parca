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

interface Props {
  isHalfScreen: boolean;
}

export const IcicleActionButtonPlaceholder = () => {
  return (
    <div className="ml-2 flex w-full flex-col items-start justify-between gap-2 md:flex-row md:items-end">
      <div>
        <label className="text-sm">Group</label>
        <div className="h-[38px] bg-[#f3f3f3] animate-pulse w-[172px]"></div>
      </div>
      <div>
        <label className="text-sm">Sort</label>
        <div className="h-[38px] bg-[#f3f3f3] animate-pulse w-[116px]"></div>
      </div>
      <div>
        <label className="text-sm">Runtimes</label>
        <div className="h-[38px] bg-[#f3f3f3] animate-pulse w-[131px]"></div>
      </div>

      <div className="h-[38px] bg-[#f3f3f3] animate-pulse w-[152px]"></div>
      <div className="h-[38px] bg-[#f3f3f3] animate-pulse w-[110px]"></div>
    </div>
  );
};

const IcicleGraphSkeleton = ({isHalfScreen}: Props) => {
  return (
    <svg
      fill="none"
      height="100%"
      viewBox="0 0 1455 688"
      width={isHalfScreen ? '1455px' : '120%'}
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
};

export default IcicleGraphSkeleton;
