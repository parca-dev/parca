import { createBrowserHistory } from 'history';
import { applyMiddleware, createStore } from 'redux';
import { composeWithDevTools } from 'redux-devtools-extension';
import { createLogger } from 'redux-logger';
import thunk from 'redux-thunk';
import rootReducer from './reducers';

const logger = (createLogger as any)();
const history = createBrowserHistory();

const dev = process.env.NODE_ENV === 'development';

let middleware = dev ? applyMiddleware(logger, thunk) :
    applyMiddleware(thunk);

if (dev) {
    middleware = composeWithDevTools(middleware);
}

export default () => {
    const store = createStore(rootReducer(history), {}, middleware);
    return { store };
};

export { history };
