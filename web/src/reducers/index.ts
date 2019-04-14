import { History } from 'history';
import { combineReducers } from 'redux';
import { Query } from '../model/model';
import { queryReducer } from './query';

export interface RootState {
    query: Query;
};

export default (history: History) => combineReducers<RootState>({
    query: queryReducer,
});
