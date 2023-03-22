import '../packages/shared/components/dist/styles.css';
import '../packages/shared/profile/dist/styles.css';

export const parameters = {
  actions: {argTypesRegex: '^on[A-Z].*'},
  controls: {
    matchers: {
      color: /(background|color)$/i,
      date: /Date$/,
    },
  },
  darkMode: {
    stylePreview: true,
    classTarget: 'html',
  },
};
