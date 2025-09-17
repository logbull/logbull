import { LoadingOutlined } from '@ant-design/icons';
import { App, Spin, Table } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import { useCallback, useEffect, useRef, useState } from 'react';

import { auditLogApi } from '../../../entity/audit-logs/api/auditLogApi';
import type { AuditLog } from '../../../entity/audit-logs/model/AuditLog';
import type { GetAuditLogsRequest } from '../../../entity/audit-logs/model/GetAuditLogsRequest';

interface Props {
  scrollContainerRef?: React.RefObject<HTMLDivElement | null>;
}

export function AuditLogsComponent({ scrollContainerRef: externalScrollRef }: Props) {
  const { message } = App.useApp();
  const [auditLogs, setAuditLogs] = useState<AuditLog[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [isLoadingMore, setIsLoadingMore] = useState(false);
  const [hasMore, setHasMore] = useState(true);
  const [total, setTotal] = useState(0);

  const pageSize = 50;

  const internalScrollRef = useRef<HTMLDivElement>(null);
  const scrollContainerRef = externalScrollRef || internalScrollRef;
  const loadingRef = useRef(false);

  useEffect(() => {
    loadAuditLogs(true);
  }, []);

  const handleScroll = useCallback(() => {
    if (!scrollContainerRef.current || isLoadingMore || !hasMore || loadingRef.current) return;

    const { scrollTop, scrollHeight, clientHeight } = scrollContainerRef.current;
    const threshold = 100;

    if (scrollHeight - scrollTop - clientHeight < threshold) {
      loadAuditLogs(false);
    }
  }, [isLoadingMore, hasMore]);

  useEffect(() => {
    const scrollContainer = scrollContainerRef.current;
    if (scrollContainer) {
      scrollContainer.addEventListener('scroll', handleScroll);
      return () => scrollContainer.removeEventListener('scroll', handleScroll);
    }
  }, [handleScroll]);

  const loadAuditLogs = async (isInitialLoad = false) => {
    if (!isInitialLoad && loadingRef.current) {
      return;
    }

    loadingRef.current = true;

    if (isInitialLoad) {
      setIsLoading(true);
      setAuditLogs([]);
    } else {
      setIsLoadingMore(true);
    }

    try {
      const offset = isInitialLoad ? 0 : auditLogs.length;
      const request: GetAuditLogsRequest = {
        limit: pageSize,
        offset: offset,
      };

      const response = await auditLogApi.getGlobalAuditLogs(request);

      if (isInitialLoad) {
        setAuditLogs(response.auditLogs);
      } else {
        setAuditLogs((prev) => {
          const existingIds = new Set(prev.map((log) => log.id));
          const newLogs = response.auditLogs.filter((log) => !existingIds.has(log.id));
          return [...prev, ...newLogs];
        });
      }

      setTotal(response.total);
      setHasMore(response.auditLogs.length === pageSize);
    } catch (error: unknown) {
      const errorMessage = error instanceof Error ? error.message : 'Failed to load audit logs';
      message.error(errorMessage);
    } finally {
      loadingRef.current = false;
      setIsLoading(false);
      setIsLoadingMore(false);
    }
  };

  const columns: ColumnsType<AuditLog> = [
    {
      title: 'User',
      dataIndex: 'userEmail',
      key: 'userEmail',
      width: 250,
      render: (userEmail: string | undefined) => (
        <span
          className={`inline-block rounded-full px-1.5 py-0.5 text-xs font-medium ${
            userEmail ? 'bg-emerald-100 text-emerald-800' : 'bg-gray-100 text-gray-600'
          }`}
        >
          {userEmail || 'System'}
        </span>
      ),
    },
    {
      title: 'Message',
      dataIndex: 'message',
      key: 'message',
      render: (message: string) => <span className="text-xs text-gray-900">{message}</span>,
    },
    {
      title: 'Project',
      dataIndex: 'projectName',
      key: 'projectName',
      width: 200,
      render: (projectId: string | undefined) => (
        <span
          className={`inline-block rounded-full px-1.5 py-0.5 text-xs font-medium ${
            projectId ? 'bg-blue-100 text-blue-800' : 'bg-gray-100 text-gray-600'
          }`}
        >
          {projectId || '-'}
        </span>
      ),
    },
    {
      title: 'Created',
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 250,
      render: (createdAt: string) => {
        return (
          <span className="text-xs text-gray-700">
            {`${dayjs(createdAt).format('MMM D, YYYY')} at ${dayjs(createdAt).format('HH:mm')} (${dayjs(createdAt).fromNow()})`}
          </span>
        );
      },
    },
  ];

  return (
    <div className="max-w-[1200px]">
      <div className="mb-4 flex items-center justify-between">
        <h1 className="text-2xl font-bold">Audit Logs</h1>
        <div className="text-sm text-gray-500">
          {isLoading ? (
            <Spin indicator={<LoadingOutlined spin />} />
          ) : (
            `${auditLogs.length} of ${total} logs`
          )}
        </div>
      </div>

      {isLoading ? (
        <div className="flex h-64 items-center justify-center">
          <Spin indicator={<LoadingOutlined spin />} size="large" />
        </div>
      ) : (
        <>
          <Table
            columns={columns}
            dataSource={auditLogs}
            pagination={false}
            rowKey="id"
            size="small"
            className="mb-4"
          />

          {isLoadingMore && (
            <div className="flex justify-center py-4">
              <Spin indicator={<LoadingOutlined spin />} />
              <span className="ml-2 text-sm text-gray-500">Loading more logs...</span>
            </div>
          )}

          {!hasMore && auditLogs.length > 0 && (
            <div className="py-4 text-center text-sm text-gray-500">
              All logs loaded ({total} total)
            </div>
          )}
        </>
      )}
    </div>
  );
}
