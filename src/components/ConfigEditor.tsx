import React, { ChangeEvent } from 'react';
import { InlineField, Input, SecretInput } from '@grafana/ui';
import { DataSourcePluginOptionsEditorProps } from '@grafana/data';
import { MyDataSourceOptions, MySecureJsonData } from '../types';

interface Props extends DataSourcePluginOptionsEditorProps<MyDataSourceOptions, MySecureJsonData> {}

export function ConfigEditor(props: Props) {
  const { onOptionsChange, options } = props;
  const { jsonData, secureJsonFields, secureJsonData } = options;

  const onEndpointChange = (event: ChangeEvent<HTMLInputElement>) => {
    onOptionsChange({
      ...options,
      jsonData: {
        ...jsonData,
        endpoint: event.target.value,
      },
    });
  };

  const onTablePrefixChange = (event: ChangeEvent<HTMLInputElement>) => {
    onOptionsChange({
      ...options,
      jsonData: {
        ...jsonData,
        tablePrefix: event.target.value,
      },
    });
  };

  const onQueryTimeoutChange = (event: ChangeEvent<HTMLInputElement>) => {
    onOptionsChange({
      ...options,
      jsonData: {
        ...jsonData,
        queryTimeoutMS: Number(event.target.value),
      },
    });
  };

  // Secure field (only sent to the backend)
  const onTokenChange = (event: ChangeEvent<HTMLInputElement>) => {
    onOptionsChange({
      ...options,
      secureJsonData: {
        token: event.target.value,
      },
    });
  };

  const onResetToken = () => {
    onOptionsChange({
      ...options,
      secureJsonFields: {
        ...options.secureJsonFields,
        token: false,
      },
      secureJsonData: {
        ...options.secureJsonData,
        token: '',
      },
    });
  };

  return (
    <>
      <InlineField label="Quack endpoint" labelWidth={16} interactive tooltip={'Quack server host:port'}>
        <Input
          id="config-editor-endpoint"
          onChange={onEndpointChange}
          value={jsonData.endpoint || 'localhost:9494'}
          placeholder="localhost:9494"
          width={40}
        />
      </InlineField>
      <InlineField label="Token" labelWidth={16} interactive tooltip={'Quack auth token (secure, backend only)'}>
        <SecretInput
          required
          id="config-editor-token"
          isConfigured={secureJsonFields.token}
          value={secureJsonData?.token}
          placeholder="Enter the Quack token"
          width={40}
          onReset={onResetToken}
          onChange={onTokenChange}
        />
      </InlineField>
      <InlineField label="Table prefix" labelWidth={16} interactive tooltip={'Optional: e.g. prd_ / uat_'}>
        <Input
          id="config-editor-table-prefix"
          onChange={onTablePrefixChange}
          value={jsonData.tablePrefix || ''}
          placeholder="Optional table prefix"
          width={40}
        />
      </InlineField>
      <InlineField label="Query timeout (ms)" labelWidth={16} interactive tooltip={'Per-query timeout'}>
        <Input
          id="config-editor-query-timeout"
          onChange={onQueryTimeoutChange}
          value={jsonData.queryTimeoutMS || 30000}
          placeholder="30000"
          width={40}
          type="number"
        />
      </InlineField>
    </>
  );
}
