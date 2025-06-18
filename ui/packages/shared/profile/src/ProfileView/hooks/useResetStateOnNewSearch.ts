import { useURLState } from "@parca/components";

export const useResetStateOnNewSearch = () => {
    const [,setCurPath] = useURLState('cur_path');

    return () => {
        setTimeout(() => {
            setCurPath(undefined);
        });
    };
}
