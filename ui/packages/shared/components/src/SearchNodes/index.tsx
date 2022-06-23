import {Input} from '@parca/components';
import {store, useAppDispatch, setSearchNodeString} from '@parca/store';
import {Provider} from 'react-redux';

const SearchNodes = () => {
  const dispatch = useAppDispatch();

  const handleChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    dispatch(setSearchNodeString(event.target.value));
  };

  return (
    <div>
      <Input className="text-sm" placeholder="Search nodes..." onChange={handleChange}></Input>
    </div>
  );
};

const SearchNodesWithProvider = () => {
  const {store: reduxStore} = store();

  return (
    <Provider store={reduxStore}>
      <SearchNodes />
    </Provider>
  );
};

export default SearchNodesWithProvider;
