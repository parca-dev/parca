
import { Typography } from '@material-ui/core';
import * as React from 'react';
import { Provider } from 'react-redux';
import { applyMiddleware } from 'redux';
import { composeWithDevTools } from 'redux-devtools-extension';
import { createLogger } from 'redux-logger';
import thunk from 'redux-thunk';
import App from './App';
import configureStore from './configureStore';

const logger = (createLogger as any)();

let middleware = applyMiddleware(logger, thunk);

if (process.env.NODE_ENV === 'development') {
    middleware = composeWithDevTools(middleware);
}

const { store } = configureStore();

function ReduxRoot() {

    return (
        <Provider store={store}>
            <App />
        </Provider>
    );
}

export default ReduxRoot;