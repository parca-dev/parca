import {createContext, ReactNode, useContext} from 'react';
import Spinner from '../Spinner';

interface ParcaThemeContextProps {
  loader: ReactNode;
}

const defaultValue: ParcaThemeContextProps = {
  loader: <Spinner />,
};

const ParcaThemeContext = createContext<ParcaThemeContextProps>(defaultValue);

export const ParcaThemeProvider = ({
  children,
  value,
}: {
  children: ReactNode;
  value?: ParcaThemeContextProps;
}) => {
  return (
    <ParcaThemeContext.Provider value={value ?? defaultValue}>
      {children}
    </ParcaThemeContext.Provider>
  );
};

export const useParcaTheme = () => {
  const context = useContext(ParcaThemeContext);
  if (context == null) {
    return defaultValue;
  }
  return context;
};

export default ParcaThemeContext;
