import Navbar from 'components/ui/Navbar';
import Head from 'next/head';
import {withRouter} from 'next/router';
import {useStore} from 'store';
import {selectUi} from 'store/ui.state';

const Header = () => {
  const {darkMode} = useStore(selectUi);
  const {setDarkMode} = useStore();

  return (
    <>
      <Head>
        <title>Parca</title>
        <link rel="icon" href="/favicon.svg" />
      </Head>
      <Navbar isDarkMode={darkMode} setDarkMode={setDarkMode} />
    </>
  );
};

export default withRouter(Header);
