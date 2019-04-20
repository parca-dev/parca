import { Action, ActionType, QueryResult } from '../model/model';

export function executeQuery(query: string) {
    return (dispatch: Function, getState: Function) => {
        api<QueryResult>('/api/v1/query_range')
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