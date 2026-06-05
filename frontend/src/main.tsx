import React, { useMemo, useState } from 'react';
import { createRoot } from 'react-dom/client';
import {
  AlertTriangle,
  CheckCircle2,
  ChevronDown,
  ChevronRight,
  Chrome,
  Clock3,
  FileText,
  Filter,
  Folder,
  HardDrive,
  Home,
  Info,
  Loader2,
  Menu,
  Monitor,
  MoreVertical,
  RefreshCw,
  Search,
  Settings,
  ShieldCheck,
  SlidersHorizontal,
  Trash2,
  X,
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

type CategoryMeta = {
  label: string;
  shortLabel: string;
  icon: React.ComponentType<{ size?: number; strokeWidth?: number }>;
};

const categoryMeta: Record<CleanCategory, CategoryMeta> = {
  user_temp: { label: '用户临时文件', shortLabel: 'User Temp', icon: Folder },
  system_temp: { label: '系统临时文件', shortLabel: 'System Temp', icon: Monitor },
  chrome_cache: { label: 'Chrome 缓存', shortLabel: 'Chrome Cache', icon: Chrome },
  edge_cache: { label: 'Edge 缓存', shortLabel: 'Edge Cache', icon: ShieldCheck },
  recycle_bin: { label: '回收站', shortLabel: 'Recycle Bin', icon: Trash2 },
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
  const [expanded, setExpanded] = useState<Set<CleanCategory>>(new Set(['user_temp']));
  const [busy, setBusy] = useState<'scan' | 'clean' | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [confirmOpen, setConfirmOpen] = useState(false);
  const [lastScanAt, setLastScanAt] = useState<Date | null>(null);

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
  const scanComplete = Boolean(scanResult && !busy);
  const issueCount = scanResult?.failures.length ?? 0;

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
      setLastScanAt(new Date());
      setSelected(new Set(result.items.filter((item) => item.defaultSelected).map((item) => item.id)));
      setExpanded(new Set(result.summaries.map((summary) => summary.category)));
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setBusy(null);
    }
  }

  async function confirmClean() {
    if (!api || !scanResult || selected.size === 0) {
      return;
    }
    setConfirmOpen(false);
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
      setBusy(null);
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
    <main className="appShell">
      <aside className="sideNav">
        <div className="brandBlock">
          <div className="brandMark">
            <ShieldCheck size={42} strokeWidth={2.6} />
          </div>
          <div>
            <h1>CleanApp</h1>
            <span>Safe Cache Cleaner</span>
          </div>
        </div>

        <nav className="navStack" aria-label="主导航">
          <button className="navItem active" type="button" title="Dashboard">
            <Home size={19} />
            <span>Dashboard</span>
          </button>
        </nav>

        <section className="navSection" aria-label="清理分类">
          <p>Cleanup Categories</p>
          <div className="categoryNav">
            {categoryOrder.map((category) => {
              const Icon = categoryMeta[category].icon;
              const items = grouped.get(category) ?? [];
              const size = items.reduce((sum, item) => sum + item.sizeBytes, 0);
              const selectedCount = items.filter((item) => selected.has(item.id)).length;
              const isActive = expanded.has(category);
              return (
                <button
                  key={category}
                  className={`categoryPill ${isActive ? 'active' : ''}`}
                  onClick={() => toggleExpanded(category)}
                  type="button"
                  title={categoryMeta[category].label}
                >
                  <Icon size={20} />
                  <span>{categoryMeta[category].shortLabel}</span>
                  <small>{items.length > 0 ? formatBytes(size) : '-'}</small>
                  <em>{selectedCount}/{items.length}</em>
                </button>
              );
            })}
          </div>
        </section>

        <div className="sideTools">
          <button className="plainNav" type="button" title="Settings">
            <Settings size={18} />
            <span>Settings</span>
          </button>
          <button className="plainNav" type="button" title="About">
            <Info size={18} />
            <span>About</span>
          </button>
        </div>

        <div className="diskCard">
          <HardDrive size={27} />
          <div>
            <strong>Local Disk (C:)</strong>
            <div className="diskBar">
              <span />
            </div>
            <small>仅扫描安全缓存目录</small>
          </div>
        </div>
      </aside>

      <section className="workspace">
        <header className="appHeader">
          <div>
            <p className="eyebrow">BQ Clean</p>
            <h2>C 盘安全缓存清理</h2>
          </div>
          <div className="headerActions">
            {busy && (
              <button className="button secondary" onClick={cancelCurrentTask} type="button" title="取消当前任务">
                <XCircle size={18} />
                <span>取消</span>
              </button>
            )}
            <button className="button primary" onClick={startScan} disabled={busy !== null || !canUseBackend} type="button" title="开始扫描">
              {busy === 'scan' ? <Loader2 className="spin" size={18} /> : <Search size={18} />}
              <span>{scanResult ? '重新扫描' : '开始扫描'}</span>
            </button>
          </div>
        </header>

        <section className="metrics" aria-label="扫描统计">
          <Metric icon={Clock3} label="可回收空间" value={formatBytes(scanResult?.totalBytes ?? 0)} note={scanComplete ? '扫描完成' : '等待扫描'} />
          <Metric icon={FileText} label="文件数量" value={`${scanResult?.totalCount ?? 0}`} note="Total files" tone="blue" />
          <Metric icon={CheckCircle2} label="已选择" value={formatBytes(selectedBytes)} note={`${selectedItems.length} files`} />
          <Metric icon={AlertTriangle} label="异常项" value={`${issueCount}`} note={issueCount > 0 ? '需要查看' : 'No critical issues'} tone={issueCount > 0 ? 'warn' : 'red'} />
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

        <section className="resultPanel" aria-label="扫描结果">
          <div className="tableToolbar">
            <label className="selectAll">
              <input
                type="checkbox"
                checked={scanResult ? scanResult.items.length > 0 && selected.size === scanResult.items.length : false}
                disabled={!scanResult || scanResult.items.length === 0}
                onChange={() => {
                  if (!scanResult) {
                    return;
                  }
                  setSelected((current) =>
                    current.size === scanResult.items.length ? new Set() : new Set(scanResult.items.map((item) => item.id)),
                  );
                }}
              />
              <span>Name</span>
            </label>
            <span>Path</span>
            <span>Risk</span>
            <span className="sizeHead">Size</span>
            <div className="toolIcons">
              <button className="iconButton" type="button" title="筛选">
                <Filter size={17} />
              </button>
              <button className="iconButton" type="button" title="视图">
                <Menu size={18} />
              </button>
            </div>
          </div>

          {!scanResult && (
            <div className="emptyState">
              <div className="emptyIcon">
                {busy === 'scan' ? <Loader2 className="spin" size={42} /> : <Search size={42} />}
              </div>
              <strong>{busy === 'scan' ? '正在扫描缓存目录' : '等待开始扫描'}</strong>
              <span>{canUseBackend ? '点击开始扫描，检查 C 盘安全缓存位置。' : '请通过 Wails 桌面应用启动以连接后端服务。'}</span>
            </div>
          )}

          {scanResult &&
            categoryOrder.map((category) => {
              const items = grouped.get(category) ?? [];
              if (items.length === 0) {
                return null;
              }
              const Icon = categoryMeta[category].icon;
              const visible = expanded.has(category);
              const allSelected = items.every((item) => selected.has(item.id));
              return (
                <section className="categoryGroup" key={category}>
                  <div className="categoryRow">
                    <button className="chevronButton" onClick={() => toggleExpanded(category)} type="button" title="展开或收起">
                      {visible ? <ChevronDown size={18} /> : <ChevronRight size={18} />}
                    </button>
                    <label className="groupName">
                      <input type="checkbox" checked={allSelected} onChange={() => toggleCategory(category, items)} />
                      <Icon size={19} />
                      <span>{categoryMeta[category].label}</span>
                    </label>
                    <span className="countText">{items.length.toLocaleString('zh-CN')} files</span>
                    <strong>{formatBytes(items.reduce((sum, item) => sum + item.sizeBytes, 0))}</strong>
                    <button className="iconButton ghost" type="button" title="更多">
                      <ChevronDown size={17} />
                    </button>
                  </div>

                  {visible && (
                    <div className="fileRows">
                      {items.slice(0, 300).map((item) => (
                        <label className="fileRow" key={item.id}>
                          <input type="checkbox" checked={selected.has(item.id)} onChange={() => toggleItem(item.id)} />
                          <FileText size={17} />
                          <span className="fileName" title={item.path}>
                            {fileName(item.path)}
                          </span>
                          <span className="path" title={item.path}>
                            {item.path}
                          </span>
                          <span className={`risk ${item.risk}`}>{item.risk === 'low' ? 'Low' : 'Medium'}</span>
                          <span className="size">{formatBytes(item.sizeBytes)}</span>
                          <MoreVertical size={17} className="moreIcon" />
                        </label>
                      ))}
                      {items.length > 300 && <div className="fileLimit">仅显示前 300 项，清理时仍按已选项目执行。</div>}
                    </div>
                  )}
                </section>
              );
            })}

          {scanResult && scanResult.failures.length > 0 && (
            <section className="failures">
              <h3>扫描异常</h3>
              {scanResult.failures.slice(0, 20).map((failure, index) => (
                <p key={`${failure.path}-${index}`} title={failure.reason}>
                  {failure.path} - {failure.reason}
                </p>
              ))}
            </section>
          )}
        </section>

        <footer className="bottomBar">
          <button className="button secondary" type="button" title="扫描选项">
            <SlidersHorizontal size={18} />
            <span>扫描选项</span>
          </button>
          <div className="scanStatus">
            <CheckCircle2 size={17} />
            <span>{scanComplete ? '扫描完成' : busy === 'scan' ? '扫描中' : '未扫描'}</span>
            {lastScanAt && <em>{lastScanAt.toLocaleString('zh-CN', { hour: '2-digit', minute: '2-digit' })}</em>}
          </div>
          <div className="bottomActions">
            <button className="button outline" onClick={startScan} disabled={busy !== null || !canUseBackend} type="button" title="重新扫描">
              <RefreshCw size={18} />
              <span>扫描</span>
            </button>
            <button
              className="button danger"
              onClick={() => setConfirmOpen(true)}
              disabled={busy !== null || selected.size === 0}
              type="button"
              title="清理所选"
            >
              {busy === 'clean' ? <Loader2 className="spin" size={18} /> : <Trash2 size={18} />}
              <span>清理所选 ({formatBytes(selectedBytes)})</span>
            </button>
          </div>
        </footer>
      </section>

      {confirmOpen && (
        <div className="modalBackdrop" role="presentation">
          <section className="confirmDialog" role="dialog" aria-modal="true" aria-label="确认清理">
            <button className="closeButton" onClick={() => setConfirmOpen(false)} type="button" title="关闭">
              <X size={18} />
            </button>
            <div className="confirmIcon">
              <Trash2 size={28} />
            </div>
            <h2>确认清理所选项目？</h2>
            <p>将清理 {selectedItems.length} 项，共 {formatBytes(selectedBytes)}。清理前会再次校验路径，失败项会保留在结果中。</p>
            <div className="dialogActions">
              <button className="button secondary" onClick={() => setConfirmOpen(false)} type="button">
                取消
              </button>
              <button className="button danger" onClick={confirmClean} type="button">
                确认清理
              </button>
            </div>
          </section>
        </div>
      )}
    </main>
  );
}

function Metric({
  icon: Icon,
  label,
  value,
  note,
  tone,
}: {
  icon: React.ComponentType<{ size?: number; strokeWidth?: number }>;
  label: string;
  value: string;
  note: string;
  tone?: 'blue' | 'warn' | 'red';
}) {
  return (
    <div className={`metric ${tone ?? ''}`}>
      <span className="metricIcon">
        <Icon size={28} strokeWidth={2.1} />
      </span>
      <div>
        <span>{label}</span>
        <strong>{value}</strong>
        <small>{note}</small>
      </div>
    </div>
  );
}

function fileName(path: string) {
  const normalized = path.replace(/\//g, '\\');
  const index = normalized.lastIndexOf('\\');
  return index >= 0 ? normalized.slice(index + 1) || normalized : normalized;
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
