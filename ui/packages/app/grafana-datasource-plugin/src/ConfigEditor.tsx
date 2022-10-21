// Copyright 2022 The Parca Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import React, {ChangeEvent, PureComponent} from 'react';
import {Field, Input, Icon} from '@grafana/ui';
import {DataSourcePluginOptionsEditorProps} from '@grafana/data';
import {ParcaDataSourceOptions} from './types';

interface Props extends DataSourcePluginOptionsEditorProps<ParcaDataSourceOptions> {}

export class ConfigEditor extends PureComponent<Props, {}> {
  onPathChange = (event: ChangeEvent<HTMLInputElement>): void => {
    const {onOptionsChange, options} = this.props;
    const jsonData = {
      ...options.jsonData,
      APIEndpoint: event.target.value,
    };
    onOptionsChange({...options, jsonData});
  };

  onResetAPIKey = (): void => {
    const {onOptionsChange, options} = this.props;
    onOptionsChange({
      ...options,
      secureJsonFields: {
        ...options.secureJsonFields,
        apiKey: false,
      },
      secureJsonData: {
        ...options.secureJsonData,
        apiKey: '',
      },
    });
  };

  render(): JSX.Element {
    const {options} = this.props;
    const {jsonData} = options;

    return (
      <div className="gf-form-group">
        <div className="gf-form"></div>
        <div className="gf-form">
          <Field label="API Endpoint" description="" required>
            <div>
              <Input
                placeholder="Parca API URL. Eg: <http://localhost:7070/api>"
                onChange={this.onPathChange}
                value={jsonData.APIEndpoint ?? ''}
                width={40}
              />
              <span>
                <br />
                <strong>Note</strong>: Please make sure cors configuration of the Parca server allow
                requests from <code>{window.location.origin}</code> origin.
                <br />
                Ensure that the Parca server is started with either{' '}
                <code>--cors-allowed-origins=&apos;{window.location.origin}&apos;</code> or{' '}
                <code>--cors-allowed-origins=&apos;*&apos;</code> flag. Please refer the{' '}
                <a
                  href="https://www.parca.dev/docs/grafana-datasource-plugin#allow-cors-requests"
                  target="_blank"
                  rel="noreferrer noopener"
                >
                  docs <Icon name="external-link-alt" />
                </a>
                .
              </span>
            </div>
          </Field>
        </div>
      </div>
    );
  }
}
