
export interface Query {
    request: QueryRequest;
    result: QueryResult;
}

export interface QueryRequest {
    expression: string;
    loading: boolean;
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
    QUERY_SUCCESS,
}

export interface Action<T> {
    type: ActionType;
    payload: T;
}