import {HasProfileDataResponse, QueryServiceClient} from '@parca/client';
import useGrpcQuery from './useGrpcQuery';

export const useHasProfileData = (client: QueryServiceClient): {loading: boolean, data: boolean, error: Error | any} => {
    const { data, loading, error } = useGrpcQuery<HasProfileDataResponse>({
        key: ['hasProfileData'],
        queryFn: async (signal) => {
            return await client.hasProfileData({}, { signal });
        },
    });

    return {loading, data: data?.hasData ?? false, error}
};
