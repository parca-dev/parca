// import 'tailwindcss/tailwind.css';
import './sb-tailwind.css';
// import '../packages/app/web/src/style/file-input.css';
// import '../packages/app/web/src/style/metrics.css';
// import '../packages/app/web/src/style/profile.css';
// import '../packages/app/web/src/style/sidenav.css';

import tailwindCss from '!style-loader!css-loader!postcss-loader!sass-loader!tailwindcss/tailwind.css';
const storybookStyles = document.createElement('style');
storybookStyles.innerHTML = tailwindCss;
document.body.appendChild(storybookStyles);

export const parameters = {
  actions: {argTypesRegex: '^on[A-Z].*'},
  darkMode: {
    darkClass: 'dark',
    lightClass: 'light',
    stylePreview: true,
  },
  options: {
    showPanel: true,
    panelPosition: 'right',
    showNav: true,
    isToolshown: true,
  },
};
