import ProfileSelector, {QuerySelection} from './ProfileSelector';
import {ProfileSelection, ProfileView} from '@parcaui/profile';
import {QueryServiceClient} from '@parcaui/client';

interface ProfileExplorerSingleProps {
  queryClient: QueryServiceClient;
  query: QuerySelection;
  selectQuery: (query: QuerySelection) => void;
  selectProfile: (source: ProfileSelection) => void;
  profile: ProfileSelection | null;
  compareProfile: () => void;
}

const ProfileExplorerSingle = ({
  queryClient,
  query,
  selectQuery,
  selectProfile,
  profile,
  compareProfile,
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
            <ProfileView queryClient={queryClient} profileSource={profile.ProfileSource()} />
          ) : (
            <></>
          )}
        </div>
      </div>
    </>
  );
};

export default ProfileExplorerSingle;
