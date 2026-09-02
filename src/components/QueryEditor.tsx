import React, { ChangeEvent } from 'react';
import { TextArea } from '@grafana/ui';
import { QueryEditorProps } from '@grafana/data';
import { DataSource } from '../datasource';
import { MyDataSourceOptions, MyQuery } from '../types';

type Props = QueryEditorProps<DataSource, MyQuery, MyDataSourceOptions>;

export function QueryEditor({ query, onChange }: Props) {
  const onSQLChange = (event: ChangeEvent<HTMLTextAreaElement>) => {
    onChange({ ...query, sql: event.target.value });
  };

  return (
    <TextArea
      value={query.sql || ''}
      onChange={onSQLChange}
      placeholder={
        'SELECT started_at AS time, count(*) AS value\nFROM monitor_tracing\nWHERE started_at > $__timeFilter(started_at)\nGROUP BY 1 ORDER BY 1'
      }
      rows={6}
    />
  );
}
