# Parca Framegraph Plugin for Grafana

## Getting started

1. Install the Parca flamegraph plugin from the [Grafana plugin repository](https://grafana.com/grafana/plugins/parca-panel/).
2. Make sure that Parca datasource plugin is installed and configured in Grafana, if not, follow the instructions [here](#configuring-the-datasource).
3. Congifure the Parca flamegraph panel:
   1. Add a new panel to your dashboard.
   2. In the Query section:
      1. Select the Parca datasource.
      2. The `Profile Type` dropdown, lists the profile types that are available on the connected Parca server. Select the profile type you want to visualize.
      3. In the `Query Selector` field, enter the query you want to visualize. The query selector is a [PromQL](https://prometheus.io/docs/prometheus/latest/querying/basics/) like expression that is used to select the profiles that you want to visualize. For example, `{job="api-server"}` will select all the profiles that have the label `job` with the value `api-server`.
   3. In the Vizualization section on the right, select the `Parca Flamegraph` visualization. Now you should be able to see the flamegraph visualization, for the selected query.
   4. Enter a suitable title for the panel in the `Panel Title` field.
   5. Save the Panel.

## Configuring the Datasource

1. Install the Parca datasource plugin from the [Grafana plugin repository](https://grafana.com/grafana/plugins/parca-datasource/).
2. In the Grafana UI, navigate to the `Configuration` -> `Data Sources` page.
3. Click on the `Add data source` button.
4. Select the `Parca` datasource.
5. Enter the API URL of the Parca server in the `API Endpoint` field. For example, `http://localhost:7070/api`.
   Note: Please make sure cors configuration of the Parca server allow requests from your Grafana Dashboard origin. If you Grafana dashboard is running at <code>http://localhost:3000</code>, then ensure that the Parca server is started with either --cors-allowed-origins='http://localhost:3000' or --cors-allowed-origins='\*' flag. Please refer the [docs](https://www.parca.dev/docs/grafana-datasource-plugin#allow-cors-requests).
6. Click on the `Save & Test` button. If the connection is successful, you should see a green `Data source is working` message.
7. Now you can use the Parca datasource in your panels.

## Screenshots

![Parca Flamegraph](https://raw.githubusercontent.com/parca-dev/parca/main/ui/packages/app/grafana-panel-plugin/src/img/screenshots/panel.png)
![Parca Flamegraph Config](https://raw.githubusercontent.com/parca-dev/parca/main/ui/packages/app/grafana-panel-plugin/src/img/screenshots/panel-config.png)
