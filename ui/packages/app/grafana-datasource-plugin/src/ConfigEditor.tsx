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
import {LegacyForms} from '@grafana/ui';
import {DataSourcePluginOptionsEditorProps} from '@grafana/data';
import {ParcaDataSourceOptions} from './types';

const {FormField} = LegacyForms;

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

  // Secure field (only sent to the backend)
  onAPIKeyChange = (event: ChangeEvent<HTMLInputElement>): void => {
    const {onOptionsChange, options} = this.props;
    onOptionsChange({
      ...options,
      secureJsonData: {
        apiKey: event.target.value,
      },
    });
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
        <div className="gf-form">
          <FormField
            label="APIEndpoint"
            labelWidth={6}
            inputWidth={26}
            onChange={this.onPathChange}
            value={jsonData.APIEndpoint ?? ''}
            placeholder="Parca API URL. Eg: <http://localhost:7070/api>"
          />
        </div>
      </div>
    );
  }
}
