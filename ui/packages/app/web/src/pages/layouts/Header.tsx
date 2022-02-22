import Navbar from '../../components/ui/Navbar';
import {useStore} from '../../store';
import {selectUi} from '../../store/ui.state';

const Header = () => {
  const {darkMode} = useStore(selectUi);
  const {setDarkMode} = useStore();

  return <Navbar isDarkMode={darkMode} setDarkMode={setDarkMode} />;
};

export default Header;
