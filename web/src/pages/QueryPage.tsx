import { Button } from '@material-ui/core';
import { makeStyles } from '@material-ui/styles';
import { withStyles, WithStyles, createStyles } from '@material-ui/core';
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
import Icon from '@material-ui/core/Icon';
import SendIcon from '@material-ui/icons/Send';
import PropTypes from 'prop-types';
import moment from 'moment';
import { ResponsiveContainer, ScatterChart, Scatter, XAxis, YAxis, ZAxis, Tooltip } from 'recharts';
import FormControlLabel from '@material-ui/core/FormControlLabel';
import Switch from '@material-ui/core/Switch';
import LinearProgress from '@material-ui/core/LinearProgress';

const styles = (theme: Theme) => createStyles({
    root: {
        flexGrow: 1,
        fontFamily: theme.typography.fontFamily,
    },
    paper: {
        margin: '20px 0px',
        padding: 10,
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
    labelSet: {
        fontFamily: 'monospace',
        padding: 10,
        fontWeight: 'bold',
    },
    noResult: {
        textAlign: 'center',
        color: theme.palette.text.secondary,
    }
});

interface Props extends RouteComponentProps<void>, WithStyles<typeof styles> {
    actions: Actions;
    query: Query;
}

interface State {
    expression: string;
    timeFrom: moment.Moment;
    timeTo: moment.Moment;
    now: boolean;
}

function renderTooltip(props: any) {
    const { active, payload } = props;

    if (active && payload && payload.length) {
        const data = payload[0].payload;

        return (
            <div style={{ backgroundColor: '#fff', border: '1px solid #999', margin: 0, padding: 10 }}>
                <p>{moment(data.timestamp).format('YYYY/M/D HH:mm')}</p>
            </div>
        );
    }

    return null;
}

function formatLabels(labels: { [key: string]: string }) {
    let tsName = (labels.__name__ || '') + '{';
    const labelStrings: string[] = [];
    for (const label in labels) {
        if (label !== '__name__') {
            labelStrings.push(label + '="' + labels[label] + '"');
        }
    }
    tsName += labelStrings.join(', ') + '}';
    return tsName;
};

class QueryPage extends React.Component<Props, State> {
    constructor(props: Props) {
        super(props);

        let search = new URLSearchParams(props.location.search);
        let expr = search.get("query") || props.query.request.expression;
        let timeFrom = search.get("from") ? moment(Number(search.get("from"))) : props.query.request.timeFrom;
        let timeTo = search.get("to") ? moment(Number(search.get("to"))) : props.query.request.timeFrom;
        let now = (search.get("now") || "").toLowerCase() != 'false';

        this.state = {
            expression: expr,
            timeFrom: timeFrom,
            timeTo: timeTo,
            now: now,
        };

        this.handleExpressionChange = this.handleExpressionChange.bind(this);
        this.handleTimeFromChange = this.handleTimeFromChange.bind(this);
        this.handleTimeToChange = this.handleTimeToChange.bind(this);
        this.handleNowChange = this.handleNowChange.bind(this);
        this.execute = this.execute.bind(this);
    }

    componentDidMount() {
        this.execute();
    }

    handleExpressionChange(event: any) {
        this.setState({
            expression: event.target.value,
            timeFrom: this.state.timeFrom,
            timeTo: this.state.timeTo,
            now: this.state.now,
        });
    }

    handleTimeFromChange(event: any) {
        this.setState({
            expression: this.state.expression,
            timeFrom: moment(event.target.value),
            timeTo: this.state.timeTo,
            now: this.state.now,
        });
    }

    handleTimeToChange(event: any) {
        this.setState({
            expression: this.state.expression,
            timeFrom: this.state.timeFrom,
            timeTo: moment(event.target.value),
            now: this.state.now,
        });
    }

    handleNowChange(event: any) {
        this.setState({
            expression: this.state.expression,
            timeFrom: this.state.timeFrom,
            timeTo: this.state.timeTo,
            now: event.target.checked,
        });
    }

    execute() {
        if(this.state.now) {
            this.props.actions.executeQuery(this.state.expression, this.state.timeFrom, moment(Date.now()));
            let q: URLSearchParams = new URLSearchParams();
            q.append("query", this.state.expression);
            q.append("from", this.state.timeFrom.valueOf().toString());
            q.append("now", "true");
            this.props.history.push({search: q.toString()});
            return
        }
        this.props.actions.executeQuery(this.state.expression, this.state.timeFrom, this.state.timeTo);
        let q: URLSearchParams = new URLSearchParams();
        q.append("query", this.state.expression);
        q.append("from", this.state.timeFrom.valueOf().toString());
        q.append("to", this.state.timeTo.valueOf().toString());
        q.append("now", "false");
        this.props.history.push({search: q.toString()});
    }

    render() {
        const { history, actions, query, classes } = this.props;

        const openProfile = (props: any) => {
            const { payload } = props;
            console.log(props);
            if (payload) {
                const data = payload;
                const q = `{${Object.entries(data.labels).map(([labelName, labelValue]) => `${labelName}="${labelValue}"`).join(",")}}`;

                window.open('/pprof/' + btoa(q) + '/' + data.timestamp + '/');
            }
        }

        return (
            <div className={classes.root}>
                <Grid container justify="center">
                    <Grid item xs={8}>
                        <Paper className={classes.expr} elevation={1}>
                            <InputBase
                                className={classes.input}
                                fullWidth
                                placeholder="Expression"
                                value={this.state.expression}
                                onChange={this.handleExpressionChange}
                                  onKeyPress={(ev: any) => {
                                      if (ev.key === 'Enter') {
                                          this.execute();
                                      }
                                  }}
                            />
                            <TextField
                                id="datetime-local-from"
                                label="From"
                                type="datetime-local"
                                value={this.state.timeFrom.format('YYYY-MM-DDTHH:mm')}
                                InputLabelProps={{
                                shrink: true,
                                }}
                                onChange={this.handleTimeFromChange}
                            />
                            <TextField
                                id="datetime-local-to"
                                label="To"
                                type="datetime-local"
                                defaultValue={this.state.timeTo.format('YYYY-MM-DDTHH:mm')}
                                InputLabelProps={{
                                shrink: true,
                                }}
                                onChange={this.handleTimeToChange}
                            />
                            <FormControlLabel
                                control={
                                <Switch
                                    checked={this.state.now}
                                    onChange={this.handleNowChange}
                                    color="primary"
                                />
                                }
                                label="Now"
                            />
                            <IconButton className={classes.iconButton} onClick={this.execute} aria-label="Search">
                                <SendIcon color="primary" />
                            </IconButton>
                        </Paper>
                    </Grid>

                    {query.result.data.map(
                        (series: Series, i: number) => {
                            return (
                                <Grid key={series.labelsetEncoded} item xs={8}>
                                    <Paper className={classes.paper}>
                                        <div className={classes.labelSet}>{formatLabels(series.labels)}</div>
                                        <div style={{ width: '100%', height: 70 }}>
                                            <ResponsiveContainer>
                                                <ScatterChart height={60} margin={{top: 10, right: 0, bottom: 0, left: 0}}>
                                                    <XAxis type="number" dataKey="timestamp" domain={['auto', 'auto']} tickFormatter={(unixTime) => moment(unixTime).format('YYYY/M/D HH:mm')} />
                                                    <YAxis type="number" dataKey="index" height={10} width={80} tick={false} tickLine={false} axisLine={false} />
                                                    <Tooltip cursor={{strokeDasharray: '3 3'}} wrapperStyle={{ zIndex: 100 }} content={renderTooltip} />
                                                    <Scatter data={series.timestamps.map((timestamp: number) => { return {labels: series.labels, timestamp: timestamp, index: 1} })} onClick={openProfile} fill='#8884d8'/>
                                                </ScatterChart>
                                            </ResponsiveContainer>
                                        </div>
                                    </Paper>
                                </Grid>
                            )
                        }
                    )}
                    {!query.request.loading && query.result.data.length == 0 &&
                        <Grid key="no-result" className={classes.noResult} item xs={8}>
                            <h3>No result</h3>
                        </Grid>
                    }
                </Grid>
            </div>
        );
    }
}

type Actions = {
    executeQuery: (query: string, fromTime: moment.Moment, toTime: moment.Moment) => void
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

export default connect(mapStateToProps, mapDispatchToProps)(withStyles(styles)(QueryPage));
