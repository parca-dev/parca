import { useURLState } from "@parca/components";

export const useResetStateOnProfileTypeChange = () => {
    const [,setGroupBy] = useURLState('group_by');
    const [,setFilterByFunction] = useURLState('filter_by_function');
    const [,setExcludeFunction] = useURLState('exclude_function');
    const [,setSearchString] = useURLState('search_string');
    const [,setCurPath] = useURLState('cur_path');

    return () => {
        setTimeout(() => {
            setGroupBy(undefined);
            setFilterByFunction(undefined);
            setExcludeFunction(undefined);
            setSearchString(undefined);
            setCurPath(undefined);
        });
    };
}
