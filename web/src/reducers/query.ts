import { Action, ActionType, Query, QueryResult } from '../model/model';
import { QuerySuccessAction } from '../actions/query';
import moment from 'moment';

const initialState: Query = {
    request: {
        expression: "",
        loading: false,
        timeFrom: moment(Date.now()).subtract(30, 'minutes'),
        timeTo: moment(Date.now()),
        now: true,
    },
    result: {
        data: [],
    },
};

export const queryReducer = (state: Query = initialState, action: Action<any>): Query => {
    switch (action.type) {
        case ActionType.QUERY_SUCCESS:
            return {
                request: {
                    expression: state.request.expression,
                    loading: false,
                    timeFrom: state.request.timeFrom,
                    timeTo: state.request.timeTo,
                    now: state.request.now,
                },
                result: (action as QuerySuccessAction).payload,
            };
            case ActionType.QUERY_FAILED:
                return initialState;
    }
    return state;
};
