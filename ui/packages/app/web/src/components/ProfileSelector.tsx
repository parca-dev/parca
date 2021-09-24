import { QueryServiceClient, ServiceError, ValuesRequest, ValuesResponse } from '@parca/client'
import { Query } from '@parca/parser'
import { ProfileSelection, timeFormatShort } from '@parca/profile'
import moment from 'moment'
import React, { useEffect, useState } from 'react'
import ProfileMetricsGraph from '../components/ProfileMetricsGraph'
import MatchersInput from './MatchersInput'
import MergeButton from './MergeButton'
import CompareButton from './CompareButton'
import Button from './ui/Button'
import ButtonGroup from './ui/ButtonGroup'
import Card from './ui/Card'
import Select from './ui/Select'

interface TimeSelection {
  from: number | null
  to: number | null
}

export interface QuerySelection {
  expression: string
  from: number
  to: number
  merge: boolean
  timeSelection: string
}

interface ProfileSelectorProps {
  queryClient: QueryServiceClient
  querySelection: QuerySelection
  selectProfile: (source: ProfileSelection) => void
  selectQuery: (query: QuerySelection) => void
  enforcedProfileName: string
  profileSelection: ProfileSelection | null
  comparing: boolean
  onCompareProfile: () => void
}

export interface ILabelValuesResult {
  response: ValuesResponse.AsObject | null
  error: ServiceError | null
}

export const useLabelValues = (
  client: QueryServiceClient,
  labelName: string
): ILabelValuesResult => {
  const [result, setResult] = useState<ILabelValuesResult>({
    response: null,
    error: null
  })

  useEffect(() => {
    const req = new ValuesRequest()
    req.setLabelName(labelName)

    client.values(req, (error: ServiceError | null, responseMessage: ValuesResponse | null) => {
      const res = responseMessage == null ? null : responseMessage.toObject()

      setResult({
        response: res,
        error: error
      })
    })
  }, [client, labelName])

  return result
}

const ProfileSelector = ({
  queryClient,
  querySelection,
  selectProfile,
  selectQuery,
  enforcedProfileName,
  profileSelection,
  comparing,
  onCompareProfile
}: ProfileSelectorProps): JSX.Element => {
  const { response, error } = useLabelValues(queryClient, '__name__')
  const profileNames =
    (error === undefined || error == null) && response !== undefined && response != null
      ? response.labelValuesList
      : []
  const profileLabels = profileNames.map(name => ({ key: name, label: name }))

  const [exactTimeSelection, setExactTimeSelection] = useState<TimeSelection>({
    from: null,
    to: null
  })
  const [timeSelection, setTimeSelection] = useState('')
  const [queryExpressionString, setQueryExpressionString] = useState(querySelection.expression)

  useEffect(() => {
    if (enforcedProfileName !== '') {
      const [q, changed] = Query.parse(querySelection.expression).setProfileName(
        enforcedProfileName
      )
      if (changed) {
        setQueryExpressionString(q.toString())
        return
      }
    }
    setQueryExpressionString(querySelection.expression)
  }, [enforcedProfileName, querySelection.expression])

  const enforcedProfileNameQuery = (): Query => {
    const pq = Query.parse(queryExpressionString)
    const [q] = pq.setProfileName(enforcedProfileName)
    return q
  }

  const query =
    enforcedProfileName !== '' ? enforcedProfileNameQuery() : Query.parse(queryExpressionString)
  const selectedProfileName = query.profileName()

  const currentFromTimeSelection = (): number => {
    if (exactTimeSelection.from != null) {
      return exactTimeSelection.from
    }
    return !isNaN(querySelection.from) ? querySelection.from : moment().utc().valueOf()
  }

  const currentToTimeSelection = (): number => {
    if (exactTimeSelection.to != null) {
      return exactTimeSelection.to
    }
    return !isNaN(querySelection.from) ? querySelection.to : moment().utc().valueOf()
  }

  const timeSelections = [
    {
      key: 'lasthour',
      label: 'Last hour',
      time: (): number[] => [
        moment().utc().subtract(1, 'hour').valueOf(),
        moment().utc().valueOf()
      ],
      relative: true
    },
    {
      key: 'lastday',
      label: 'Last day',
      time: (): number[] => [moment().utc().subtract(1, 'day').valueOf(), moment().utc().valueOf()],
      relative: true
    },
    {
      key: 'last3days',
      label: 'Last 3 days',
      time: (): number[] => [
        moment().utc().subtract(3, 'days').valueOf(),
        moment().utc().valueOf()
      ],
      relative: true
    },
    {
      key: 'last7days',
      label: 'Last 7 days',
      time: (): number[] => [
        moment().utc().subtract(7, 'days').valueOf(),
        moment().utc().valueOf()
      ],
      relative: true
    },
    {
      key: 'last14days',
      label: 'Last 14 days',
      time: (): number[] => [
        moment().utc().subtract(14, 'days').valueOf(),
        moment().utc().valueOf()
      ],
      relative: true
    },
    {
      key: 'custom',
      label: (
        <a>
          {moment(currentFromTimeSelection()).utc().format(timeFormatShort)} &rArr;{' '}
          {moment(currentToTimeSelection()).utc().format(timeFormatShort)}
        </a>
      ),
      time: (): number[] => [
        moment(currentFromTimeSelection()).utc().valueOf(),
        moment(currentToTimeSelection()).utc().valueOf()
      ],
      relative: false
    }
  ]
  const timePresets = timeSelections
    .filter(selection => selection.relative)
    .map(selection => ({ key: selection.key, label: selection.label as string }))

  const timeSelectionByKey = (key: string): number => timeSelections.findIndex(e => e.key === key)

  const currentTimeSelection = (): string => {
    if (timeSelection !== '') {
      return timeSelection
    }
    if (querySelection.timeSelection !== undefined) {
      return querySelection.timeSelection
    }
    return 'lasthour'
  }

  const setNewQueryExpression = (expr: string, merge: boolean): void => {
    const ts = timeSelectionByKey(currentTimeSelection())
    const [from, to] = timeSelections[ts].time()
    selectQuery({
      expression: expr,
      from: from,
      to: to,
      merge: merge,
      timeSelection: timeSelections[ts].key
    })
  }

  const setQueryExpression = (): void => {
    setNewQueryExpression(query.toString(), false)
  }

  const addLabelMatcher = (key: string, value: string): void => {
    const [newQuery, changed] = Query.parse(queryExpressionString).setMatcher(key, value)
    if (changed) {
      setNewQueryExpression(newQuery.toString(), false)
    }
  }

  const setMergedSelection = (): void => {
    setNewQueryExpression(queryExpressionString, true)
  }

  const setMatchersString = (matchers: string): void => {
    const newExpressionString = `${selectedProfileName}{${matchers}}`
    setQueryExpressionString(newExpressionString)
  }

  const setTimeRange = (from: number, to: number): void => {
    setTimeSelection('custom')
    setExactTimeSelection({
      from: from,
      to: to
    })
  }

  const setProfileName = (profileName: string): void => {
    const [newQuery, changed] = query.setProfileName(profileName)
    if (changed) {
      setQueryExpressionString(newQuery.toString())
    }
  }

  const handleCompareClick = (): void => onCompareProfile()

  const searchDisabled =
    queryExpressionString === undefined ||
    queryExpressionString === '' ||
    queryExpressionString === '{}'

  const mergeDisabled = selectedProfileName === '' || querySelection.expression === undefined
  const compareDisabled = selectedProfileName === '' || querySelection.expression === undefined

  return (
    <>
      <Card>
        <Card.Header>
          <div className='flex space-x-4'>
            <Select
              items={profileLabels}
              selectedKey={selectedProfileName}
              onSelection={setProfileName}
              placeholder='Select profile...'
            />
            <MatchersInput
              queryClient={queryClient}
              setMatchersString={setMatchersString}
              runQuery={setQueryExpression}
              currentQuery={query}
            />
            <Select
              items={timePresets}
              selectedKey={currentTimeSelection()}
              onSelection={key => setTimeSelection(key ?? '')}
            />
            {searchDisabled
              ? (
                <div>
                  <Button disabled={true}>Search</Button>
                </div>
                )
              : (
                  <>
                    <ButtonGroup style={{ marginRight: 5 }}>
                      <MergeButton disabled={mergeDisabled} onClick={setMergedSelection} />
                      {!comparing && (
                          <CompareButton disabled={compareDisabled} onClick={handleCompareClick} />
                      )}
                    </ButtonGroup>
                    <div>
                      <Button
                        onClick={(e: React.MouseEvent<HTMLElement>) => {
                          e.preventDefault()
                          setQueryExpression()
                        }}
                      >
                        Search
                      </Button>
                    </div>
                  </>
                )}
          </div>
        </Card.Header>
        {!querySelection.merge && (
          <Card.Body>
            {querySelection.expression !== undefined &&
            querySelection.expression.length > 0 &&
            querySelection.from !== undefined &&
            querySelection.to !== undefined &&
            (profileSelection == null || profileSelection.Type() !== 'merge')
              ? (
                <ProfileMetricsGraph
                  queryClient={queryClient}
                  queryExpression={querySelection.expression}
                  from={querySelection.from}
                  to={querySelection.to}
                  select={selectProfile}
                  profile={profileSelection}
                  setTimeRange={(from: number, to: number) => {
                    setTimeRange(from, to)
                    selectQuery({
                      expression: queryExpressionString,
                      from: from,
                      to: to,
                      merge: false,
                      timeSelection: 'custom'
                    })
                  }}
                  addLabelMatcher={addLabelMatcher}
                />
                )
              : (
                <>
                  {(profileSelection == null || profileSelection.Type() !== 'merge') && (
                    <div className='my-20 text-center'>
                      <p>Run a query, and the result will be displayed here.</p>
                    </div>
                  )}
                </>
                )
            }
          </Card.Body>
        )}
      </Card>
    </>
  )
}

export default ProfileSelector
