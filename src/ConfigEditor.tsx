import React from 'react';
import { Field, FieldSet, Input, SecretInput } from '@grafana/ui';
import { DataSourcePluginOptionsEditorProps } from '@grafana/data';

import {
  DuckDBDataSourceOptions,
  DuckDBSecureOptions,
} from './types';

type Props = DataSourcePluginOptionsEditorProps<DuckDBDataSourceOptions, DuckDBSecureOptions>;

export class ConfigEditor extends React.PureComponent<Props> {
  onEndpointChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    this.props.onOptionsChange({
      ...this.props.options,
      jsonData: { ...this.props.options.jsonData, endpoint: e.currentTarget.value },
    });
  };

  onPrefixChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    this.props.onOptionsChange({
      ...this.props.options,
      jsonData: { ...this.props.options.jsonData, tablePrefix: e.currentTarget.value },
    });
  };

  onTimeoutChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    this.props.onOptionsChange({
      ...this.props.options,
      jsonData: { ...this.props.options.jsonData, queryTimeoutMS: Number(e.currentTarget.value) },
    });
  };

  onTokenChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    this.props.onOptionsChange({
      ...this.props.options,
      secureJsonData: { ...this.props.options.secureJsonData, token: e.currentTarget.value },
    });
  };

  onTokenReset = () => {
    this.props.onOptionsChange({
      ...this.props.options,
      secureJsonData: {},
      secureJsonFields: { ...this.props.options.secureJsonFields, token: false },
    });
  };

  render() {
    const { options } = this.props;
    const { jsonData, secureJsonFields, secureJsonData } = options;

    return (
      <FieldSet label="DuckDB Quack">
        <Field label="Quack endpoint" description="Quack server host:port (e.g. localhost:9494)">
          <Input value={jsonData?.endpoint || 'localhost:9494'} onChange={this.onEndpointChange} />
        </Field>
        <Field label="Token" description="Quack auth token (stored securely)">
          <SecretInput
            isConfigured={Boolean(secureJsonFields?.token)}
            value={secureJsonData?.token || ''}
            onChange={this.onTokenChange}
            onReset={this.onTokenReset}
          />
        </Field>
        <Field label="Table prefix" description="可选：默认查无前缀表；填 prd_ / uat_ 后可在 SQL 中省略前缀">
          <Input value={jsonData?.tablePrefix || ''} onChange={this.onPrefixChange} />
        </Field>
        <Field label="Query timeout (ms)" description="单查询超时，默认 30000">
          <Input
            type="number"
            value={jsonData?.queryTimeoutMS || 30000}
            onChange={this.onTimeoutChange}
          />
        </Field>
      </FieldSet>
    );
  }
}
