import { Action, ActionType, Query, QueryResult } from '../model/model';
import { QuerySuccessAction } from '../actions/query';

const initialState: Query = {
    request: {
        expression: "",
        loading: false,
    },
    result: {
            series: [],
    },
};

export const queryReducer = (state: Query = initialState, action: Action<any>): Query => {
    switch (action.type) {
        case ActionType.QUERY_SUCCESS:
            return {
                request: {
                    expression: state.request.expression,
                    loading: false,
                },
                result: (action as QuerySuccessAction).payload,
            };
    }
    return state;
};
