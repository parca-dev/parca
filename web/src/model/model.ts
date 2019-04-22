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
    series: Series[];
}

export interface Series {
    labelset: string;
    labelsetEncoded: string;
    timestamps: number[];
}

export enum ActionType {
    QUERY_STARTED,
    QUERY_SUCCESS,
}

export interface Action<T> {
    type: ActionType;
    payload: T;
}