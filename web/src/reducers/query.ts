import { Action, ActionType, Query, QueryResult } from '../model/model';
import { QuerySuccessAction } from '../actions/query';

const initialState: Query = {
    request: {
        expression: "",
        loading: false,
    },
    result: {
            series: [{
                labelset: "{name=\"test\"}",
                timestamps: [
                    1555269198,
                    1555269298,
                    1555269398,
                    1555269498,
                ],
            }],
    },
};

export const queryReducer = (state: Query = initialState, action: Action<any>): Query => {
    switch (action.type) {
        case ActionType.QUERY_SUCCESS:
            state.result = (action as QuerySuccessAction).payload
            return state;
    }
    return state;
};
