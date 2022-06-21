import {Input} from '@parca/components';
import {useAppDispatch, setSearchNodeString} from '@parca/store';

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

export default SearchNodes;
