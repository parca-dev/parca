import moment from 'moment';

export interface Query {
    request: QueryRequest;
    result: QueryResult;
}

export interface QueryRequest {
    expression: string;
    loading: boolean;
    timeFrom: moment.Moment;
    timeTo: moment.Moment;
    now: boolean;
}

export interface QueryResult {
    data: QueryResultData;
}

export interface QueryResultData {
    series: Series[];
}

export interface Labels {
  [key: string]: string;
}

export interface Series {
    labels: Labels;
    labelsetEncoded: string;
    timestamps: number[];
}

export enum ActionType {
    QUERY_STARTED,
    QUERY_SUCCESS,
    QUERY_FAILED,
}

export interface Action<T> {
    type: ActionType;
    payload: T;
}
