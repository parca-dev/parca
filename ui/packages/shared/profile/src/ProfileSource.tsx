import React from 'react'
import moment from 'moment'
import { Query } from '@parca/parser'
import { Label, QueryRequest, ProfileDiffSelection, SingleProfile, MergeProfile, DiffProfile } from '@parca/client'
import { Timestamp } from 'google-protobuf/google/protobuf/timestamp_pb'

export interface ProfileSource {
  QueryRequest: () => QueryRequest
  DiffSelection: () => ProfileDiffSelection
  Describe: () => JSX.Element
  toString: () => string
}

export interface ProfileSelection {
  ProfileName: () => string
  HistoryParams: () => { [key: string]: any }
  ProfileSource: () => ProfileSource
  Type: () => string
}

export const timeFormat = 'MMM D, [at] h:mma [(UTC)]'
export const timeFormatShort = 'MMM D, h:mma'

export function ParamsString (params: { [key: string]: string }): string {
  return Object.keys(params).map(function (key) { return `${key}=${params[key]}` }).join('&')
}

export function SuffixParams (params: { [key: string]: any }, suffix: string): { [key: string]: any } {
  return Object.fromEntries(
    Object.entries(params).map(([key, value]) => [`${key}${suffix}`, value])
  )
}

export function ParseLabels (labels: string[]): Label.AsObject[] {
  return labels.map(function(labelString): Label.AsObject {
    const parts = labelString.split('=', 2)
    return {name: parts[0], value: parts[1]}
  })
}

export function ProfileSelectionFromParams (
  expression: string,
  from: string,
  to: string,
  merge: string,
  labels: string[],
  time: string
): ProfileSelection | null {
  if (merge && merge == 'true' && from && to && expression) {
    return new MergedProfileSelection(parseInt(from), parseInt(to), expression)
  }
  if (labels && time) {
    return new SingleProfileSelection(ParseLabels(labels), parseInt(time))
  }
  return null
}

export class SingleProfileSelection implements ProfileSelection {
  labels: Label.AsObject[]
  time: number

  constructor (labels: Label.AsObject[], time: number) {
    this.labels = labels
    this.time = time
  }

  ProfileName (): string {
    const label = this.labels.find((e) => e.name == '__name__')
    return label !== undefined ? label.value : ''
  }

  HistoryParams (): { [key: string]: any } {
    return {
      labels: this.labels.map((label) => `${label.name}=${label.value}`),
      time: this.time
    }
  }

  Type (): string {
    return 'single'
  }

  ProfileSource (): ProfileSource {
    return new SingleProfileSource(
      this.labels,
      this.time
    )
  }
}

export class MergedProfileSelection implements ProfileSelection {
  from: number
  to: number
  query: string

  constructor (from: number, to: number, query: string) {
    this.from = from
    this.to = to
    this.query = query
  }

  ProfileName (): string {
    return Query.parse(this.query).profileName()
  }

  HistoryParams (): { [key: string]: string } {
    return {
      mode: 'merge',
      from: this.from.toString(),
      to: this.to.toString(),
      query: this.query
    }
  }

  Type (): string {
    return 'merge'
  }

  ProfileSource (): ProfileSource {
    return new MergedProfileSource(
      this.from,
      this.to,
      this.query
    )
  }
}

export class SingleProfileSource implements ProfileSource {
  labels: Label.AsObject[]
  time: number

  constructor (labels: Label.AsObject[], time: number) {
    this.labels = labels
    this.time = time
  }

  query (): string {
    const seriesQuery = this.labels.reduce(function(agg: string, label: Label.AsObject) {
      return agg + `${label.name}="${label.value}",`
    }, '{')
    return seriesQuery + '}'
  }

  DiffSelection (): ProfileDiffSelection {
    const sel = new ProfileDiffSelection()
    sel.setMode(ProfileDiffSelection.Mode.MODE_SINGLE_UNSPECIFIED)

    const singleProfile = new SingleProfile()
    const ts = new Timestamp()
    ts.fromDate(moment(this.time).toDate())
    singleProfile.setTime(ts)
    singleProfile.setQuery(this.query())
    sel.setSingle(singleProfile)

    return sel
  }

  QueryRequest (): QueryRequest {
    const req = new QueryRequest()
    req.setMode(QueryRequest.Mode.MODE_SINGLE_UNSPECIFIED)
    const singleQueryRequest = new SingleProfile()
    const ts = new Timestamp()
    ts.fromDate(moment(this.time).toDate())
    singleQueryRequest.setTime(ts)
    singleQueryRequest.setQuery(this.query())
    req.setSingle(singleQueryRequest)
    return req
  }

  profileName (): string {
    const label = this.labels.find((e) => e.name == '__name__')
    return label !== undefined ? label.value : ''
  }

  Describe (): JSX.Element {
    const profileName = this.profileName()
    return (
      <>
        <p>
          {profileName != '' ? <a>{profileName} profile of </a> : ''}{'  '}
          {this.labels.filter(label => (label.name != '__name__')).map((label) => (
              <button
                  key={label.name}
                  type="button"
                  className="inline-block rounded-lg text-gray-700 bg-gray-200 dark:bg-gray-700 dark:text-gray-400 px-2 py-1 text-xs font-bold mr-3"
              >
                  {`${label.name}="${label.value}"`}
              </button>
          ))}
        </p>
        <p>
          {moment(this.time).utc().format(timeFormat)}
        </p>
      </>
    )
  }

  stringLabels (): string[] {
    return Object.keys(this.labels).filter(key => (key != '__name__')).map((k) => `${k}=${this.labels[k]}`)
  }

  toString (): string {
    return `single profile of type ${this.profileName()} with labels ${this.stringLabels().join(', ')} collected at ${moment(this.time).utc().format(timeFormat)}`
  }
}

export class ProfileDiffSource implements ProfileSource {
  a: ProfileSource
  b: ProfileSource

  constructor (a: ProfileSource, b: ProfileSource) {
    this.a = a
    this.b = b
  }

  DiffSelection (): ProfileDiffSelection {
    return new ProfileDiffSelection()
  }

  QueryRequest (): QueryRequest {
    const req = new QueryRequest()
    req.setMode(QueryRequest.Mode.MODE_DIFF)
    const diffQueryRequest = new DiffProfile()

    diffQueryRequest.setA(this.a.DiffSelection())
    diffQueryRequest.setB(this.b.DiffSelection())

    req.setDiff(diffQueryRequest)
    return req
  }

  Describe (): JSX.Element {
    return (
      <>
        <p>Browse the comparison</p>
      </>
    )
  }

  toString (): string {
    return `${this.a.toString()} compared with ${this.b.toString()}`
  }
}

export class MergedProfileSource implements ProfileSource {
  from: number
  to: number
  query: string

  constructor (from: number, to: number, query: string) {
    this.from = from
    this.to = to
    this.query = query
  }

  DiffSelection (): ProfileDiffSelection {
    const sel = new ProfileDiffSelection()
    sel.setMode(ProfileDiffSelection.Mode.MODE_MERGE)

    const mergeProfile = new MergeProfile()

    const startTs = new Timestamp()
    startTs.fromDate(moment(this.from).toDate())
    mergeProfile.setStart(startTs)

    const endTs = new Timestamp()
    endTs.fromDate(moment(this.to).toDate())
    mergeProfile.setEnd(endTs)

    mergeProfile.setQuery(this.query)

    sel.setMerge(mergeProfile)

    return sel
  }

  QueryRequest (): QueryRequest {
    const req = new QueryRequest()
    req.setMode(QueryRequest.Mode.MODE_MERGE)

    const mergeQueryRequest = new MergeProfile()

    const startTs = new Timestamp()
    startTs.fromDate(moment(this.from).toDate())
    mergeQueryRequest.setStart(startTs)

    const endTs = new Timestamp()
    endTs.fromDate(moment(this.to).toDate())
    mergeQueryRequest.setEnd(endTs)

    mergeQueryRequest.setQuery(this.query)

    req.setMerge(mergeQueryRequest)

    return req
  }

  Describe (): JSX.Element {
    return (
      <a>Merge of "{this.query}" from {moment(this.from).utc().format(timeFormat)} to {moment(this.to).utc().format(timeFormat)}</a>
    )
  }

  toString (): string {
    return `merged profiles of query "${this.query}" from ${moment(this.from).utc().format(timeFormat)} to ${moment(this.to).utc().format(timeFormat)}`
  }
}
