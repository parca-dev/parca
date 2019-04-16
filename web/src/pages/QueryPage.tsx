import { Button } from '@material-ui/core';
import { makeStyles } from '@material-ui/styles';
import { Theme } from '@material-ui/core/styles';
import * as React from 'react';
import { connect } from 'react-redux';
import * as redux from 'redux';
import { RouteComponentProps } from 'react-router-dom';
import TextField from '@material-ui/core/TextField';
import Grid from '@material-ui/core/Grid';
import Paper from '@material-ui/core/Paper';
import { GridSpacing } from '@material-ui/core/Grid';
import { RootState } from '../reducers/index';
import { executeQuery } from '../actions/query';
import { Query, Series } from '../model/model';
import InputBase from '@material-ui/core/InputBase';
import Divider from '@material-ui/core/Divider';
import IconButton from '@material-ui/core/IconButton';
import MenuIcon from '@material-ui/icons/Menu';
import SearchIcon from '@material-ui/icons/Search';
import DirectionsIcon from '@material-ui/icons/Directions';
import PropTypes from 'prop-types';

const useStyles = makeStyles((theme: Theme) => ({
    root: {
        flexGrow: 1,
    },
    paper: {
        padding: 10,
        textAlign: 'center',
    },
    expr: {
        margin: '20px 0px',
        padding: '2px 4px',
        display: 'flex',
        alignItems: 'center',
    },
    input: {
        marginLeft: 8,
        flex: 1,
    },
    iconButton: {
        padding: 10,
    },
}));

interface Props extends RouteComponentProps<void> {
    actions: Actions;
    query: Query;
}

function QueryPage(props: Props) {
    const classes = useStyles();
    const { actions, query } = props;

    return (
        <div className={classes.root}>
            <Grid container justify="center">
                <Grid item xs={8}>
                    <Paper className={classes.expr} elevation={1}>
                        <InputBase className={classes.input} fullWidth placeholder="Expression" />
                        <IconButton className={classes.iconButton} onClick={() => actions.executeQuery("")} aria-label="Search">
                            <SearchIcon />
                        </IconButton>
                    </Paper>
                </Grid>

                {query.result.series.map(
                (series: Series) => {
                return (
                <Grid item xs={8}>
                    <Paper className={classes.paper}>
                        <h4>{series.labelset}</h4>
                        <ul>
                            {series.timestamps.map(
                            (timestamp: number) => {
                            return (
                            <li><a href={series.labelset + '/' + timestamp}>{timestamp}</a></li>
                            )
                            }
                            )}
                        </ul>
                    </Paper>
                </Grid>
                )
                }
                )}
            </Grid>
        </div>
    );
}

type Actions = {
    executeQuery: (query: string) => void
}

type Dispatch = {
    actions: Actions;
}

function mapStateToProps(state: RootState) {
  return {
    query: state.query,
  };
}

function mapDispatchToProps(dispatch: redux.Dispatch<redux.AnyAction>): Dispatch {
  return {
      actions: redux.bindActionCreators({executeQuery}, dispatch),
  };
}

export default connect(mapStateToProps, mapDispatchToProps)(QueryPage);

