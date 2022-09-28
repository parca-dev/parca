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

import defaults from 'lodash/defaults';

import React, { useCallback, useState, useEffect } from 'react';
import { Field, Input, Select } from '@grafana/ui';
import { QueryEditorProps, SelectableValue } from '@grafana/data';
import { DataSource } from './datasource';
import { defaultQuery, ParcaDataSourceOptions, ParcaQuery } from './types';
import { wellKnownProfiles } from '@parca/profile/src/ProfileTypeSelector';

type Props = QueryEditorProps<DataSource, ParcaQuery, ParcaDataSourceOptions>;

const getDropDownItemForProfileKey = (key: string) => {
  if (wellKnownProfiles[key]) {
    return {
      value: key,
      label: wellKnownProfiles[key].name,
      description: wellKnownProfiles[key].help,
    };
  } else {
    return {
      value: key,
      label: key,
    };
  }
};

export const QueryEditor = (props: Props) => {
  const query = defaults(props.query, defaultQuery);
  const { parcaQuery } = query;

  const [profileType, setProfileType] = useState<SelectableValue<string>>(() => {
    const indexOf = (parcaQuery || '').indexOf('{');
    if (!parcaQuery || indexOf === -1) {
      return null;
    }
    const profileType = parcaQuery.slice(0, indexOf);
    return getDropDownItemForProfileKey(profileType);
  });
  const [querySelector, setQuerySelector] = useState<string>(() => {
    const indexOf = (parcaQuery || '').indexOf('{');
    if (!parcaQuery || indexOf === -1) {
      return '';
    }
    return parcaQuery.slice(indexOf);
  });
  const [profileTypesLoading, setProfileTypesLoading] = useState<boolean>(true);
  const [profileTypes, setProfileTypes] = useState<SelectableValue<string>[]>([]);
  const onParcaQueryChange = useCallback((parcaQuery: string) => {
    const { onChange, query, onRunQuery } = props;
    if (parcaQuery === query.parcaQuery) {
      return;
    }
    console.log('parcaQuery', parcaQuery);
    onChange({ ...query, parcaQuery });
    // executes the query
    onRunQuery();
  }, []);

  useEffect(() => {
    (async () => {
      try {
        const { datasource } = props;
        const { response } = await datasource.queryClient.profileTypes({});
        const profileNames = response.types
          .map(
            (type) =>
              `${type.name}:${type.sampleType}:${type.sampleUnit}:${type.periodType}:${type.periodUnit}${
                type.delta ? ':delta' : ''
              }`
          )
          .sort((a: string, b: string): number => {
            return a.localeCompare(b);
          });
        const profileTypes = profileNames.map(getDropDownItemForProfileKey);
        setProfileTypes(profileTypes);
      } catch (error) {
        console.log('error', error);
      }
      setProfileTypesLoading(false);
    })();
  }, []);

  useEffect(() => {
    if (!profileType || !querySelector) {
      return;
    }
    const parcaQuery = `${profileType.value}${querySelector}`;

    onParcaQueryChange(parcaQuery);
  }, [profileType, querySelector]);

  return (
    <div>
      <div className="gf-form">
        <Field label="Profile Type" description="" required>
          <Select
            options={profileTypes}
            value={profileType}
            onChange={setProfileType}
            isLoading={profileTypesLoading}
            width={40}
            placeholder="Select a profile type"
          />
        </Field>
      </div>
      <div className="gf-form">
        <Field label="Query Selector" description="" required>
          <Input
            placeholder='podName="api"'
            onChange={(e) => setQuerySelector((e.target as HTMLInputElement).value)}
            value={querySelector}
          />
        </Field>
      </div>
    </div>
  );
};
