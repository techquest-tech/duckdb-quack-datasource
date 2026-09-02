import React from 'react';
import { TextArea } from '@grafana/ui';
import { QueryEditorProps } from '@grafana/data';

import { DuckDBDatasource } from './datasource';
import { DuckDBDataSourceOptions, DuckDBQuery } from './types';

type Props = QueryEditorProps<DuckDBDatasource, DuckDBQuery, DuckDBDataSourceOptions>;

export class QueryEditor extends React.PureComponent<Props> {
  onChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    this.props.onChange({ ...this.props.query, sql: e.currentTarget.value });
  };

  render() {
    const { query } = this.props;
    return (
      <TextArea
        value={query.sql || ''}
        onChange={this.onChange}
        placeholder={'SELECT started_at AS time, count(*) AS value\nFROM uat_monitor_tracing\nWHERE started_at > $__timeFilter(started_at)\nGROUP BY 1 ORDER BY 1'}
        rows={6}
      />
    );
  }
}
