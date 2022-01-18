import ProfileSelector, {QuerySelection} from './ProfileSelector';
import {ProfileDiffSource, ProfileSelection, ProfileView} from '@parcaui/profile';
import {Query} from '@parcaui/parser';
import {QueryServiceClient} from '@parcaui/client';

interface ProfileExplorerCompareProps {
  queryClient: QueryServiceClient;

  queryA: QuerySelection;
  queryB: QuerySelection;
  profileA: ProfileSelection | null;
  profileB: ProfileSelection | null;
  selectQueryA: (query: QuerySelection) => void;
  selectQueryB: (query: QuerySelection) => void;
  selectProfileA: (source: ProfileSelection) => void;
  selectProfileB: (source: ProfileSelection) => void;
}

const ProfileExplorerCompare = ({
  queryClient,
  queryA,
  queryB,
  profileA,
  profileB,
  selectQueryA,
  selectQueryB,
  selectProfileA,
  selectProfileB,
}: ProfileExplorerCompareProps): JSX.Element => {
  return (
    <>
      <div className="grid grid-cols-2">
        <div className="pr-2">
          <ProfileSelector
            queryClient={queryClient}
            querySelection={queryA}
            profileSelection={profileA}
            selectProfile={selectProfileA}
            selectQuery={selectQueryA}
            enforcedProfileName={''}
            comparing={true}
            onCompareProfile={() => {}}
          />
        </div>
        <div className="pl-2">
          <ProfileSelector
            queryClient={queryClient}
            querySelection={queryB}
            profileSelection={profileB}
            selectProfile={selectProfileB}
            selectQuery={selectQueryB}
            enforcedProfileName={Query.parse(queryA.expression).profileName()}
            comparing={true}
            onCompareProfile={() => {}}
          />
        </div>
      </div>
      <div className="grid grid-cols-1">
        {profileA != null && profileB != null ? (
          <ProfileView
            queryClient={queryClient}
            profileSource={
              new ProfileDiffSource(profileA.ProfileSource(), profileB.ProfileSource())
            }
          />
        ) : (
          <div>
            <div className="my-20 text-center">
              <p>Select a profile on both sides.</p>
            </div>
          </div>
        )}
      </div>
    </>
  );
};

export default ProfileExplorerCompare;
