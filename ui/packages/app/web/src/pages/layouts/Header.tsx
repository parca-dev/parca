import Navbar from '../../components/ui/Navbar';
import Head from 'next/head';
import {useStore} from '../../store';
import {selectUi} from '../../store/ui.state';

const Header = () => {
  const {darkMode} = useStore(selectUi);
  const {setDarkMode} = useStore();

  return (
    <>
      {/* Todo: replace with react-helmet */}
      {/* <Head>
        <title>Parca</title>
        <link rel="icon" href="/favicon.svg" />
      </Head> */}
      <Navbar isDarkMode={darkMode} setDarkMode={setDarkMode} />
    </>
  );
};

export default Header;
