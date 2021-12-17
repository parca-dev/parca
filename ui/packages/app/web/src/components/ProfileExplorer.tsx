import { QuerySelection } from './ProfileSelector'
import { ProfileSelection, ProfileSelectionFromParams, SuffixParams } from '@parca/profile'
import { NextRouter, withRouter } from 'next/router'
import ProfileExplorerSingle from './ProfileExplorerSingle'
import ProfileExplorerCompare from './ProfileExplorerCompare'
import { QueryServiceClient } from '@parca/client'

interface ProfileExplorerProps {
  router: NextRouter
  queryClient: QueryServiceClient
}

const ProfileExplorer = ({
  router,
  queryClient
}: ProfileExplorerProps): JSX.Element => {
  /* eslint-disable */
  // Disable eslint due to params being snake case
  const {
    expression_a,
    from_a,
    to_a,
    merge_a,
    labels_a,
    time_a,
    time_selection_a,
    compare_a,
    expression_b,
    from_b,
    to_b,
    merge_b,
    labels_b,
    time_b,
    time_selection_b,
    compare_b
  } = router.query
  /* eslint-enable */

  const queryParams = Object.fromEntries(Object.entries(router.query))

  const filterSuffix = (o: { [key: string]: (string | string[] | undefined) }, suffix: string): { [key: string]: (string | string[] | undefined) } =>
    Object.fromEntries(
      Object.entries(o).filter(([key]) => !key.endsWith(suffix))
    )

  const selectProfileA = async (p: ProfileSelection): Promise<boolean> => {
    return await router.push({
      pathname: '/',
      query: { ...queryParams, ...SuffixParams(p.HistoryParams(), '_a') }
    })
  }

  const selectProfileB = async (p: ProfileSelection): Promise<boolean> => {
    return await router.push({
      pathname: '/',
      query: { ...queryParams, ...SuffixParams(p.HistoryParams(), '_b') }
    })
  }

  // Show the SingleProfileExplorer when not comparing
  if (compare_a !== 'true' && compare_b !== 'true') {
    const query = {
      expression: expression_a as string,
      from: parseInt(from_a as string),
      to: parseInt(to_a as string),
      merge: (merge_a as string) === 'true',
      timeSelection: time_selection_a as string
    }
    const profile = ProfileSelectionFromParams(
      expression_a as string,
      from_a as string,
      to_a as string,
      merge_a as string,
      labels_a as string[],
      time_a as string
    )
    const selectQuery = async (q: QuerySelection): Promise<boolean> => {
      return await router.push({
        pathname: document.location.pathname,
        // Filtering the _a suffix causes us to reset potential profile
        // selection when running a new query.
        query: {
          ...filterSuffix(queryParams, '_a'),
          ...{
            expression_a: q.expression,
            from_a: q.from.toString(),
            to_a: q.to.toString(),
            merge_a: q.merge,
            time_selection_a: q.timeSelection
          }
        }
      })
    }
    const selectProfile = async (p: ProfileSelection): Promise<boolean> => {
      return await router.push({
        pathname: '/',
        query: { ...queryParams, ...SuffixParams(p.HistoryParams(), '_a') }
      })
    }

    const compareProfile = (): void => {
      let compareQuery = {
        compare_a: 'true',
        expression_a: query.expression,
        from_a: query.from.toString(),
        to_a: query.to.toString(),
        merge_a: query.merge,
        time_selection_a: query.timeSelection,

        compare_b: 'true',
        expression_b: query.expression,
        from_b: query.from.toString(),
        to_b: query.to.toString(),
        merge_b: query.merge,
        time_selection_b: query.timeSelection
      }

      if (profile != null) {
        compareQuery = {
          ...SuffixParams(profile.HistoryParams(), '_a'),
          ...compareQuery
        }
      }

      void router.push({
        pathname: '/',
        query: compareQuery
      })
    }

    return (
      <ProfileExplorerSingle
        queryClient={queryClient}
        query={query}
        profile={profile}
        selectQuery={selectQuery}
        selectProfile={selectProfile}
        compareProfile={compareProfile}
      />
    )
  }

  const queryA = {
    expression: expression_a as string,
    from: parseInt(from_a as string),
    to: parseInt(to_a as string),
    merge: (merge_a as string) === 'true',
    timeSelection: time_selection_a as string
  }
  const queryB = {
    expression: expression_b as string,
    from: parseInt(from_b as string),
    to: parseInt(to_b as string),
    merge: (merge_b as string) === 'true',
    timeSelection: time_selection_b as string
  }

  const profileA = ProfileSelectionFromParams(
    expression_a as string,
    from_a as string,
    to_a as string,
    merge_a as string,
    labels_a as string[],
    time_a as string
  )
  const profileB = ProfileSelectionFromParams(
    expression_b as string,
    from_b as string,
    to_b as string,
    merge_b as string,
    labels_b as string[],
    time_b as string
  )

  const selectQueryA = async (q: QuerySelection): Promise<boolean> => {
    return await router.push({
      pathname: '/',
      // Filtering the _a suffix causes us to reset potential profile
      // selection when running a new query.
      query: {
        ...filterSuffix(queryParams, '_a'),
        ...{
          compare_a: 'true',
          expression_a: q.expression,
          from_a: q.from.toString(),
          to_a: q.to.toString(),
          merge_a: q.merge,
          time_selection_a: q.timeSelection
        }
      }
    })
  }

  const selectQueryB = async (q: QuerySelection): Promise<boolean> => {
    return await router.push({
      pathname: '/',
      // Filtering the _b suffix causes us to reset potential profile
      // selection when running a new query.
      query: {
        ...filterSuffix(queryParams, '_b'),
        ...{
          compare_b: 'true',
          expression_b: q.expression,
          from_b: q.from.toString(),
          to_b: q.to.toString(),
          merge_b: q.merge,
          time_selection_b: q.timeSelection
        }
      }
    })
  }

  return (
    <ProfileExplorerCompare
      queryClient={queryClient}
      queryA={queryA}
      queryB={queryB}
      profileA={profileA}
      profileB={profileB}
      selectQueryA={selectQueryA}
      selectQueryB={selectQueryB}
      selectProfileA={selectProfileA}
      selectProfileB={selectProfileB}
    />
  )
}

export default withRouter(ProfileExplorer)
