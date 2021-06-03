import { AppBar, Toolbar, Typography, withWidth } from '@material-ui/core';
import { Theme, makeStyles, createStyles } from '@material-ui/core/styles';
import { isWidthUp, WithWidth } from '@material-ui/core/withWidth';
import * as React from 'react';
import { connect } from 'react-redux';
import { Route, RouteComponentProps, Router } from 'react-router-dom';
import { history } from './configureStore';
import withRoot from './withRoot';
import QueryPage from './pages/QueryPage';
import { RootState } from './reducers/index';

function Routes() {
    const classes = useStyles();
    const pathPrefix = window.location.pathname === '/' ? '' : window.location.pathname;

    return (
        <div className={classes.content}>
            <Route
                render={(props) => (
                    <QueryPage {...props} pathPrefix={pathPrefix} />
                )}
            />
        </div>
    );
}

interface Props extends RouteComponentProps<void>, WithWidth {
}

function App(props?: Props) {
    const classes = useStyles();

    if (!props) {
        return null;
    }

    return (
        <Router history={history}>
            <div className={classes.root}>
                <div className={classes.appFrame}>
                    <AppBar className={classes.appBar}>
                        <Toolbar>
                            <Typography variant="h6" color="inherit" noWrap={isWidthUp('sm', props.width)}>
                                Conprof
                            </Typography>
                        </Toolbar>
                    </AppBar>
                    <Routes />
                </div>
            </div>
        </Router>
    );

}

const useStyles = makeStyles(theme => ({
    root: {
        width: '100%',
        zIndex: 1,
        overflow: 'hidden',
    },
    appFrame: {
        position: 'relative',
        display: 'flex',
        width: '100%',
        height: '100%',
    },
    appBar: props => ({
        zIndex: theme.zIndex.drawer + 1,
        position: 'absolute',
    }),
    navIconHide: {
        [theme.breakpoints.up('md')]: {
            display: 'none',
        },
    },
    content: {
        backgroundColor: theme.palette.background.default,
        width: '100%',
        height: 'calc(100% - 56px)',
        marginTop: 56,
        [theme.breakpoints.up('sm')]: {
            height: 'calc(100% - 64px)',
            marginTop: 64,
        },
    },
}));

function mapStateToProps(state: RootState) {
    return {
    };
}

export default connect(mapStateToProps)(withRoot(withWidth()(App)));
