import { Action, ActionType, QueryResult } from '../model/model';
import * as moment from 'moment';

export function executeQuery(query: string, fromTime: moment.Moment, toTime: moment.Moment) {
    return (dispatch: Function, getState: Function) => {
        api<QueryResult>('/api/v1/query_range?from='+fromTime.valueOf()+'&to='+toTime.valueOf()+'&query='+encodeURIComponent(query))
            .then((result) => {
                dispatch({ type: ActionType.QUERY_SUCCESS, payload: result });
            })
            .catch(error => {
                /* show error message */
            })
    };
}

function api<T>(url: string): Promise<T> {
    return fetch(url)
    .then(response => {
        if (!response.ok) {
            throw new Error(response.statusText)
        }
        return response.json().then(data => data as T);
    })
}

export interface QuerySuccessAction extends Action<any> {
    payload: QueryResult;
};

export interface QueryStartedAction extends Action<any> {
};