import React, { useMemo, useState } from 'react';
import { createRoot } from 'react-dom/client';
import {
  AlertTriangle,
  CheckCircle2,
  ChevronDown,
  ChevronRight,
  Loader2,
  RefreshCw,
  Search,
  ShieldCheck,
  Trash2,
  XCircle,
} from 'lucide-react';
import './styles.css';

type CleanCategory = 'user_temp' | 'system_temp' | 'chrome_cache' | 'edge_cache' | 'recycle_bin';
type RiskLevel = 'low' | 'medium';

type ScanItem = {
  id: string;
  path: string;
  sizeBytes: number;
  modifiedAt: string;
  category: CleanCategory;
  risk: RiskLevel;
  defaultSelected: boolean;
  isDirectory: boolean;
  isVirtual: boolean;
};

type CategorySummary = {
  category: CleanCategory;
  itemCount: number;
  sizeBytes: number;
};

type ScanFailure = {
  path: string;
  reason: string;
};

type ScanResult = {
  taskId: string;
  items: ScanItem[];
  summaries: CategorySummary[];
  totalCount: number;
  totalBytes: number;
  failures: ScanFailure[];
  cancelled: boolean;
};

type CleanResult = {
  deletedCount: number;
  deletedBytes: number;
  skippedCount: number;
  failures: ScanFailure[];
  cancelled: boolean;
};

type WailsAPI = {
  Scan: (options: { categories: CleanCategory[] }) => Promise<ScanResult>;
  Clean: (request: { taskId: string; itemIds: string[] }) => Promise<CleanResult>;
  CancelTask: (taskId: string) => Promise<void>;
};

const categoryLabels: Record<CleanCategory, string> = {
  user_temp: '用户临时文件',
  system_temp: '系统临时文件',
  chrome_cache: 'Chrome 缓存',
  edge_cache: 'Edge 缓存',
  recycle_bin: '回收站',
};

const categoryOrder: CleanCategory[] = [
  'user_temp',
  'system_temp',
  'chrome_cache',
  'edge_cache',
  'recycle_bin',
];

declare global {
  interface Window {
    go?: {
      main?: {
        App?: WailsAPI;
      };
    };
  }
}

function App() {
  const api = window.go?.main?.App;
  const [scanResult, setScanResult] = useState<ScanResult | null>(null);
  const [cleanResult, setCleanResult] = useState<CleanResult | null>(null);
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [expanded, setExpanded] = useState<Set<CleanCategory>>(new Set(['user_temp', 'system_temp']));
  const [busy, setBusy] = useState<'scan' | 'clean' | null>(null);
  const [error, setError] = useState<string | null>(null);

  const grouped = useMemo(() => {
    const groups = new Map<CleanCategory, ScanItem[]>();
    for (const category of categoryOrder) {
      groups.set(category, []);
    }
    for (const item of scanResult?.items ?? []) {
      groups.get(item.category)?.push(item);
    }
    for (const items of groups.values()) {
      items.sort((a, b) => b.sizeBytes - a.sizeBytes);
    }
    return groups;
  }, [scanResult]);

  const selectedItems = useMemo(() => {
    return (scanResult?.items ?? []).filter((item) => selected.has(item.id));
  }, [scanResult, selected]);

  const selectedBytes = selectedItems.reduce((sum, item) => sum + item.sizeBytes, 0);
  const canUseBackend = Boolean(api);

  async function startScan() {
    if (!api) {
      setError('Wails 运行时不可用，请通过桌面应用启动。');
      return;
    }
    setBusy('scan');
    setError(null);
    setCleanResult(null);
    try {
      const result = await api.Scan({ categories: [] });
      setScanResult(result);
      const defaults = new Set(result.items.filter((item) => item.defaultSelected).map((item) => item.id));
      setSelected(defaults);
      setExpanded(new Set(result.summaries.map((summary) => summary.category)));
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setBusy(null);
    }
  }

  async function startClean() {
    if (!api || !scanResult || selected.size === 0) {
      return;
    }
    const confirmed = window.confirm(`确认清理 ${selected.size} 项，共 ${formatBytes(selectedBytes)}？`);
    if (!confirmed) {
      return;
    }
    setBusy('clean');
    setError(null);
    try {
      const result = await api.Clean({ taskId: scanResult.taskId, itemIds: Array.from(selected) });
      setCleanResult(result);
      const failedPaths = new Set(result.failures.map((failure) => failure.path));
      const remaining = new Set(
        scanResult.items
          .filter((item) => selected.has(item.id) && failedPaths.has(item.path))
          .map((item) => item.id),
      );
      setSelected(remaining);
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setBusy(null);
    }
  }

  async function cancelCurrentTask() {
    if (!api || !scanResult?.taskId) {
      return;
    }
    try {
      await api.CancelTask(scanResult.taskId);
    } catch {
      setBusy(null);
    }
  }

  function toggleItem(itemID: string) {
    setSelected((current) => {
      const next = new Set(current);
      if (next.has(itemID)) {
        next.delete(itemID);
      } else {
        next.add(itemID);
      }
      return next;
    });
  }

  function toggleCategory(category: CleanCategory, items: ScanItem[]) {
    setSelected((current) => {
      const next = new Set(current);
      const allSelected = items.length > 0 && items.every((item) => next.has(item.id));
      for (const item of items) {
        if (allSelected) {
          next.delete(item.id);
        } else {
          next.add(item.id);
        }
      }
      return next;
    });
  }

  function toggleExpanded(category: CleanCategory) {
    setExpanded((current) => {
      const next = new Set(current);
      if (next.has(category)) {
        next.delete(category);
      } else {
        next.add(category);
      }
      return next;
    });
  }

  return (
    <main className="shell">
      <header className="topbar">
        <div className="brand">
          <ShieldCheck aria-hidden="true" />
          <div>
            <h1>CleanApp</h1>
            <span>Windows 缓存清理</span>
          </div>
        </div>
        <div className="actions">
          {busy && (
            <button className="button secondary" onClick={cancelCurrentTask} title="取消当前任务">
              <XCircle size={18} />
              <span>取消</span>
            </button>
          )}
          <button className="button primary" onClick={startScan} disabled={busy !== null || !canUseBackend} title="开始扫描">
            {busy === 'scan' ? <Loader2 className="spin" size={18} /> : <Search size={18} />}
            <span>{scanResult ? '重新扫描' : '扫描'}</span>
          </button>
        </div>
      </header>

      <section className="metrics" aria-label="扫描统计">
        <Metric label="可清理空间" value={formatBytes(scanResult?.totalBytes ?? 0)} />
        <Metric label="文件数量" value={`${scanResult?.totalCount ?? 0}`} />
        <Metric label="已选择" value={formatBytes(selectedBytes)} />
        <Metric label="异常" value={`${scanResult?.failures.length ?? 0}`} tone={(scanResult?.failures.length ?? 0) > 0 ? 'warn' : 'ok'} />
      </section>

      {error && (
        <section className="notice error">
          <AlertTriangle size={18} />
          <span>{error}</span>
        </section>
      )}

      {cleanResult && (
        <section className="notice success">
          <CheckCircle2 size={18} />
          <span>
            已清理 {formatBytes(cleanResult.deletedBytes)}，删除 {cleanResult.deletedCount} 项，跳过 {cleanResult.skippedCount} 项
          </span>
        </section>
      )}

      <section className="content">
        <aside className="sidebar">
          {categoryOrder.map((category) => {
            const items = grouped.get(category) ?? [];
            const size = items.reduce((sum, item) => sum + item.sizeBytes, 0);
            const count = items.length;
            const selectedCount = items.filter((item) => selected.has(item.id)).length;
            return (
              <button
                key={category}
                className="categoryButton"
                onClick={() => toggleExpanded(category)}
                title={categoryLabels[category]}
              >
                <span>{expanded.has(category) ? <ChevronDown size={18} /> : <ChevronRight size={18} />}</span>
                <strong>{categoryLabels[category]}</strong>
                <small>
                  {selectedCount}/{count} · {formatBytes(size)}
                </small>
              </button>
            );
          })}
        </aside>

        <section className="results" aria-label="扫描结果">
          {!scanResult && (
            <div className="emptyState">
              <Search size={44} />
              <strong>{busy === 'scan' ? '正在扫描' : '等待扫描'}</strong>
            </div>
          )}

          {scanResult &&
            categoryOrder.map((category) => {
              const items = grouped.get(category) ?? [];
              if (items.length === 0) {
                return null;
              }
              const visible = expanded.has(category);
              const allSelected = items.every((item) => selected.has(item.id));
              return (
                <section className="categorySection" key={category}>
                  <div className="categoryHeader">
                    <button className="iconButton" onClick={() => toggleExpanded(category)} title="展开或收起">
                      {visible ? <ChevronDown size={18} /> : <ChevronRight size={18} />}
                    </button>
                    <label className="checkRow">
                      <input type="checkbox" checked={allSelected} onChange={() => toggleCategory(category, items)} />
                      <span>{categoryLabels[category]}</span>
                    </label>
                    <span>{formatBytes(items.reduce((sum, item) => sum + item.sizeBytes, 0))}</span>
                  </div>

                  {visible && (
                    <div className="table">
                      {items.slice(0, 300).map((item) => (
                        <label className="row" key={item.id}>
                          <input type="checkbox" checked={selected.has(item.id)} onChange={() => toggleItem(item.id)} />
                          <span className="path" title={item.path}>
                            {item.path}
                          </span>
                          <span className={`risk ${item.risk}`}>{item.risk === 'low' ? '低风险' : '中风险'}</span>
                          <span className="size">{formatBytes(item.sizeBytes)}</span>
                        </label>
                      ))}
                      {items.length > 300 && <div className="row muted">仅显示前 300 项</div>}
                    </div>
                  )}
                </section>
              );
            })}

          {scanResult && scanResult.failures.length > 0 && (
            <section className="failures">
              <h2>异常</h2>
              {scanResult.failures.slice(0, 20).map((failure, index) => (
                <p key={`${failure.path}-${index}`} title={failure.reason}>
                  {failure.path} · {failure.reason}
                </p>
              ))}
            </section>
          )}
        </section>
      </section>

      <footer className="footer">
        <button className="button secondary" onClick={startScan} disabled={busy !== null || !canUseBackend} title="重新扫描">
          <RefreshCw size={18} />
          <span>刷新</span>
        </button>
        <button className="button danger" onClick={startClean} disabled={busy !== null || selected.size === 0} title="清理所选">
          {busy === 'clean' ? <Loader2 className="spin" size={18} /> : <Trash2 size={18} />}
          <span>清理所选</span>
        </button>
      </footer>
    </main>
  );
}

function Metric({ label, value, tone }: { label: string; value: string; tone?: 'ok' | 'warn' }) {
  return (
    <div className={`metric ${tone ?? ''}`}>
      <span>{label}</span>
      <strong>{value}</strong>
    </div>
  );
}

function formatBytes(bytes: number) {
  if (!Number.isFinite(bytes) || bytes <= 0) {
    return '0 B';
  }
  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  let value = bytes;
  let unit = 0;
  while (value >= 1024 && unit < units.length - 1) {
    value /= 1024;
    unit++;
  }
  return `${value >= 10 || unit === 0 ? value.toFixed(0) : value.toFixed(1)} ${units[unit]}`;
}

function errorMessage(err: unknown) {
  if (err instanceof Error) {
    return err.message;
  }
  return String(err);
}

createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
);

