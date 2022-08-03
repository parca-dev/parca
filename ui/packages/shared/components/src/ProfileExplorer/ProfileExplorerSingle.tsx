import {QueryServiceClient} from '@parca/client';
import {ProfileSelection, ProfileViewWithData} from '@parca/profile';

import {NavigateFunction} from '../ProfileExplorer';
import ProfileSelector, {QuerySelection} from '../ProfileSelector';
interface ProfileExplorerSingleProps {
  queryClient: QueryServiceClient;
  query: QuerySelection;
  selectQuery: (query: QuerySelection) => void;
  selectProfile: (source: ProfileSelection) => void;
  profile: ProfileSelection | null;
  compareProfile: () => void;
  navigateTo: NavigateFunction;
}

const ProfileExplorerSingle = ({
  queryClient,
  query,
  selectQuery,
  selectProfile,
  profile,
  compareProfile,
  navigateTo,
}: ProfileExplorerSingleProps): JSX.Element => {
  return (
    <>
      <div className="grid grid-cols-1">
        <div>
          <ProfileSelector
            queryClient={queryClient}
            querySelection={query}
            selectQuery={selectQuery}
            selectProfile={selectProfile}
            closeProfile={() => {}}
            profileSelection={profile}
            comparing={false}
            onCompareProfile={compareProfile}
            enforcedProfileName={''} // TODO
          />
        </div>
      </div>
      <div className="grid grid-cols-1">
        <div>
          {profile != null ? (
            <ProfileViewWithData
              queryClient={queryClient}
              profileSource={profile.ProfileSource()}
              navigateTo={navigateTo}
            />
          ) : (
            <></>
          )}
        </div>
      </div>
    </>
  );
};

export default ProfileExplorerSingle;
