import * as React from 'react';
import { Provider } from 'react-redux';
import App from './App';
import configureStore from './configureStore';

const { store } = configureStore();

function ReduxRoot() {

    return (
        <Provider store={store}>
            <App />
        </Provider>
    );
}

export default ReduxRoot;