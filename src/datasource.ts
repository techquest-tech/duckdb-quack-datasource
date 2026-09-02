import {
  DataQueryRequest,
  DataQueryResponse,
  DataSourceApi,
  DataSourceInstanceSettings,
} from '@grafana/data';
import { getBackendSrv } from '@grafana/runtime';

import {
  DuckDBDataSourceOptions,
  DuckDBQuery,
  DuckDBSecureOptions,
} from './types';

export class DuckDBDatasource extends DataSourceApi<DuckDBQuery, DuckDBDataSourceOptions> {
  constructor(instanceSettings: DataSourceInstanceSettings<DuckDBDataSourceOptions>) {
    super(instanceSettings);
  }

  // 后端插件：查询由插件进程（Go + duckcall）经 Quack 直连执行，前端无需处理。
  async query(request: DataQueryRequest<DuckDBQuery>): Promise<DataQueryResponse> {
    return { data: [] };
  }

  async testDatasource(): Promise<{ status: string; message: string }> {
    const res = (await getBackendSrv().get(`/api/datasources/uid/${this.uid}/health`)) as {
      status?: string;
      message?: string;
    };
    if (res && res.status === 'OK') {
      return { status: 'success', message: res.message ?? 'Quack connection OK' };
    }
    return { status: 'error', message: res?.message ?? 'Quack connection failed' };
  }
}
