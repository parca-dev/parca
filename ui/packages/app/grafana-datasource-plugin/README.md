# Parca Datasource Plugin for Grafana

## Getting started

1. Install the Parca datasource plugin from the [Grafana plugin repository](https://grafana.com/grafana/plugins/parca-datasource/).
2. In the Grafana UI, navigate to the `Configuration` -> `Data Sources` page.
3. Click on the `Add data source` button.
4. Select the `Parca` datasource.
5. Enter the API URL of the Parca server in the `API Endpoint` field. For example, `http://localhost:7070/api`.
   Note: Please make sure cors configuration of the Parca server allow requests from your Grafana Dashboard origin. If you Grafana dashboard is running at `http://localhost:3000`, then ensure that the Parca server is started with either `--cors-allowed-origins='http://localhost:3000'` or `--cors-allowed-origins='\*'` flag. Please refer the [docs](https://www.parca.dev/docs/grafana-datasource-plugin#allow-cors-requests).
6. Click on the `Save & Test` button. If the connection is successful, you should see a green `Data source is working` message.
7. Now you can use the Parca datasource in your panels.

## Screeenshot

![Parca Datasource Plugin](https://raw.githubusercontent.com/parca-dev/parca/main/ui/packages/app/grafana-datasource-plugin/src/img/screenshots/datasource-config.png)
