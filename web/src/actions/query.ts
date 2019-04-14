import { Action, ActionType, QueryResult } from '../model/model';

export function executeQuery(query: string) {
    return (dispatch: Function, getState: Function) => {
        let result: QueryResult = {
            series: [{
                labelset: "{name=\"test1\"}",
                timestamps: [
                    1555269198,
                    1555269298,
                    1555269398,
                    1555269498,
                ],
            }],
        };
        dispatch({ type: ActionType.QUERY_SUCCESS, payload: result });
    };
}

export interface QuerySuccessAction extends Action<any> {
    payload: QueryResult;
};