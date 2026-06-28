import React, { useMemo, useRef, useState } from 'react';
import { createRoot } from 'react-dom/client';
import {
  AlertTriangle,
  BarChart3,
  Camera,
  CheckCircle2,
  ChevronDown,
  ChevronRight,
  Code2,
  Chrome,
  Clock3,
  Database,
  Download,
  ExternalLink,
  FileText,
  Filter,
  FoldVertical,
  Folder,
  FolderTree,
  UnfoldVertical,
  HardDrive,
  Home,
  Info,
  Loader2,
  Menu,
  Monitor,
  MoreVertical,
  Package,
  RefreshCw,
  Search,
  Settings,
  ShieldCheck,
  SlidersHorizontal,
  Trash2,
  TrendingDown,
  TrendingUp,
  X,
  XCircle,
} from 'lucide-react';
import './styles.css';

type CleanCategory = 'user_temp' | 'system_temp' | 'chrome_cache' | 'edge_cache' | 'edge_indexeddb' | 'vscode_cache' | 'windows_cache' | 'dev_cache' | 'windows_update' | 'windows_logs' | 'app_cache' | 'recycle_bin';
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

type DiskSnapshot = {
  id: string;
  createdAt: string;
  label: string;
  drive: string;
  entries: { path: string; sizeBytes: number; fileCount: number }[];
  totalBytes: number;
};

type SnapshotInfo = {
  id: string;
  createdAt: string;
  label: string;
  totalBytes: number;
  entryCount: number;
};

type SnapshotDiff = {
  path: string;
  oldSize: number;
  newSize: number;
  deltaBytes: number;
  deltaPercent: number;
  cleanable: boolean;
};

type SnapshotCompareResult = {
  oldSnapshotId: string;
  newSnapshotId: string;
  oldLabel: string;
  newLabel: string;
  oldTotalBytes: number;
  newTotalBytes: number;
  diffs: SnapshotDiff[];
};

type SnapshotPathCompareResult = {
  oldSnapshotId: string;
  newSnapshotId: string;
  path: string;
  diffs: SnapshotDiff[];
};

const detailGrowthMinBytes = 20 * 1024 * 1024;
// Deeper drill-down levels use a lower threshold since nested directories
// grow by smaller amounts; otherwise deep folders would be hidden entirely.
const childGrowthMinBytes = 2 * 1024 * 1024;

function isDetailGrowthDiff(diff: SnapshotDiff) {
  return diff.deltaBytes > detailGrowthMinBytes;
}

function isChildDetailGrowthDiff(diff: SnapshotDiff) {
  return diff.deltaBytes > childGrowthMinBytes;
}

type WailsAPI = {
  Scan: (options: { categories: CleanCategory[] }) => Promise<ScanResult>;
  Clean: (request: { taskId: string; itemIds: string[] }) => Promise<CleanResult>;
  CleanGrowthPaths: (request: { snapshotId: string; paths: string[] }) => Promise<CleanResult>;
  TakeSnapshot: (drive: string, label: string) => Promise<DiskSnapshot>;
  ListSnapshots: () => Promise<SnapshotInfo[]>;
  CompareSnapshots: (oldID: string, newID: string) => Promise<SnapshotCompareResult>;
  CompareSnapshotPath: (oldID: string, newID: string, path: string) => Promise<SnapshotPathCompareResult>;
  DeleteSnapshot: (id: string) => Promise<void>;
  OpenInExplorer: (path: string) => Promise<void>;
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
  edge_indexeddb: { label: 'Edge IndexedDB 大文件 (>50M)', shortLabel: 'Edge IndexedDB', icon: Database },
  vscode_cache: { label: 'VS Code 扩展缓存', shortLabel: 'VS Code Cache', icon: Code2 },
  windows_cache: { label: 'Windows 系统缓存', shortLabel: 'Win Cache', icon: HardDrive },
  dev_cache: { label: '开发者工具缓存', shortLabel: 'Dev Cache', icon: Package },
  windows_update: { label: '系统更新与日志', shortLabel: 'Win Update', icon: RefreshCw },
  windows_logs: { label: '系统遥测日志 (WMI)', shortLabel: 'WMI Logs', icon: FileText },
  app_cache: { label: '第三方应用缓存', shortLabel: 'App Cache', icon: Download },
  recycle_bin: { label: '回收站', shortLabel: 'Recycle Bin', icon: Trash2 },
};

const categoryOrder: CleanCategory[] = [
  'user_temp',
  'system_temp',
  'chrome_cache',
  'edge_cache',
  'edge_indexeddb',
  'vscode_cache',
  'windows_cache',
  'dev_cache',
  'windows_update',
  'windows_logs',
  'app_cache',
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
  const [activeView, setActiveView] = useState<'cache' | 'growth'>('cache');
  const [scanResult, setScanResult] = useState<ScanResult | null>(null);
  const [cleanResult, setCleanResult] = useState<CleanResult | null>(null);
  const [snapshots, setSnapshots] = useState<SnapshotInfo[]>([]);
  const [compareResult, setCompareResult] = useState<SnapshotCompareResult | null>(null);
  const [growthCleanResult, setGrowthCleanResult] = useState<CleanResult | null>(null);
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [growthSelected, setGrowthSelected] = useState<Set<string>>(new Set());
  const [expandedDiffs, setExpandedDiffs] = useState<Set<string>>(new Set());
  const [childDiffs, setChildDiffs] = useState<Record<string, SnapshotDiff[]>>({});
  const [loadingChildPaths, setLoadingChildPaths] = useState<Set<string>>(new Set());
  const inFlightChildPaths = useRef(new Set<string>());
  const [expanded, setExpanded] = useState<Set<CleanCategory>>(new Set(['user_temp']));
  const [busy, setBusy] = useState<'scan' | 'clean' | 'growth' | 'growthClean' | 'snapshot' | 'list' | 'compare' | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [confirmOpen, setConfirmOpen] = useState(false);
  const [growthConfirmOpen, setGrowthConfirmOpen] = useState(false);
  const [aboutOpen, setAboutOpen] = useState(false);
  const [lastScanAt, setLastScanAt] = useState<Date | null>(null);
  const [lastGrowthScanAt, setLastGrowthScanAt] = useState<Date | null>(null);
  const [labelInput, setLabelInput] = useState('');
  const [oldSnapshotID, setOldSnapshotID] = useState('');
  const [newSnapshotID, setNewSnapshotID] = useState('');

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

  const selectedBytes = useMemo(
    () => selectedItems.reduce((sum, item) => sum + item.sizeBytes, 0),
    [selectedItems],
  );

  const { snapshotDiffs, growthEntries, shrinkEntries, totalComparedGrowth, totalComparedShrink } = useMemo(() => {
    const diffs = (compareResult?.diffs ?? []).filter(isDetailGrowthDiff);
    const growth = diffs.filter((diff) => diff.deltaBytes > 0);
    const shrink = diffs.filter((diff) => diff.deltaBytes < 0);
    return {
      snapshotDiffs: diffs,
      growthEntries: growth,
      shrinkEntries: shrink,
      totalComparedGrowth: growth.reduce((sum, entry) => sum + entry.deltaBytes, 0),
      totalComparedShrink: shrink.reduce((sum, entry) => sum + Math.abs(entry.deltaBytes), 0),
    };
  }, [compareResult]);

  const { selectedGrowthEntries, selectedGrowthBytes } = useMemo(() => {
    const entries = snapshotDiffs.filter((entry) => growthSelected.has(entry.path) && entry.cleanable && entry.deltaBytes > 0);
    return {
      selectedGrowthEntries: entries,
      selectedGrowthBytes: entries.reduce((sum, entry) => sum + Math.max(entry.newSize, 0), 0),
    };
  }, [snapshotDiffs, growthSelected]);

  const categoryStats = useMemo(() => {
    const stats = new Map<CleanCategory, { sizeBytes: number; selectedCount: number; allSelected: boolean }>();
    for (const [category, items] of grouped) {
      let sizeBytes = 0;
      let selectedCount = 0;
      for (const item of items) {
        sizeBytes += item.sizeBytes;
        if (selected.has(item.id)) {
          selectedCount++;
        }
      }
      stats.set(category, {
        sizeBytes,
        selectedCount,
        allSelected: items.length > 0 && selectedCount === items.length,
      });
    }
    return stats;
  }, [grouped, selected]);
  const canUseBackend = Boolean(api);
  const scanComplete = Boolean(scanResult && !busy);
  const issueCount = scanResult?.failures.length ?? 0;
  const aboutTime = useMemo(() => new Date().toLocaleString('zh-CN'), [aboutOpen]);

  async function startScan() {
    if (!api) {
      setError('Wails 运行时不可用，请通过桌面应用启动。');
      return;
    }
    setBusy('scan');
    setError(null);
    setCleanResult(null);
    try {
      const result = normalizeScanResult(await api.Scan({ categories: [] }));
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
      const result = normalizeCleanResult(await api.Clean({ taskId: scanResult.taskId, itemIds: Array.from(selected) }));
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

  async function startGrowthScan() {
    if (!api) {
      setError('Wails 运行时不可用，请通过桌面应用启动。');
      return;
    }
    setBusy('snapshot');
    setError(null);
    setGrowthCleanResult(null);
    try {
      const label = labelInput.trim() || new Date().toLocaleString('zh-CN');
      await api.TakeSnapshot('', label);
      setLabelInput('');
      setLastGrowthScanAt(new Date());
      await loadSnapshots();
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setBusy(null);
    }
  }

  async function loadSnapshots() {
    if (!api) {
      return;
    }
    setBusy('list');
    setError(null);
    try {
      const list = await api.ListSnapshots();
      setSnapshots(list ?? []);
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setBusy(null);
    }
  }

  async function compareSnapshots() {
    if (!api || !oldSnapshotID || !newSnapshotID) {
      return;
    }
    if (oldSnapshotID === newSnapshotID) {
      setError('请选择两个不同的快照进行对比。');
      return;
    }
    setBusy('compare');
    setError(null);
    setGrowthCleanResult(null);
    inFlightChildPaths.current.clear();
    try {
      const result = normalizeSnapshotCompareResult(await api.CompareSnapshots(oldSnapshotID, newSnapshotID));
      setCompareResult(result);
      setGrowthSelected(new Set(result.diffs.filter((diff) => diff.cleanable && isDetailGrowthDiff(diff)).map((diff) => diff.path)));
      setChildDiffs({});
      setExpandedDiffs(new Set(result.diffs.filter(isDetailGrowthDiff).slice(0, 5).map((diff) => diff.path)));
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setBusy(null);
    }
  }

  async function deleteSnapshot(id: string) {
    if (!api) {
      return;
    }
    try {
      await api.DeleteSnapshot(id);
      if (oldSnapshotID === id) {
        setOldSnapshotID('');
      }
      if (newSnapshotID === id) {
        setNewSnapshotID('');
      }
      await loadSnapshots();
    } catch (err) {
      setError(errorMessage(err));
    }
  }

  function openExplorer(path: string) {
    api?.OpenInExplorer(path).catch((err) => setError(errorMessage(err)));
  }

  async function confirmGrowthClean() {
    if (!api || !compareResult || selectedGrowthEntries.length === 0) {
      return;
    }
    setGrowthConfirmOpen(false);
    setBusy('growthClean');
    setError(null);
    try {
      const result = normalizeCleanResult(
        await api.CleanGrowthPaths({
          snapshotId: compareResult.newSnapshotId,
          paths: selectedGrowthEntries.map((entry) => entry.path),
        }),
      );
      setGrowthCleanResult(result);
      const failedPaths = new Set(result.failures.map((failure) => failure.path));
      setGrowthSelected(new Set(selectedGrowthEntries.filter((entry) => failedPaths.has(entry.path)).map((entry) => entry.path)));
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

  function expandAllCategories() {
    setExpanded(new Set(categoryOrder.filter((category) => (grouped.get(category)?.length ?? 0) > 0)));
  }

  function collapseAllCategories() {
    setExpanded(new Set());
  }

  const allCategoriesExpanded = useMemo(() => {
    const nonEmpty = categoryOrder.filter((category) => (grouped.get(category)?.length ?? 0) > 0);
    return nonEmpty.length > 0 && nonEmpty.every((category) => expanded.has(category));
  }, [grouped, expanded]);

  function toggleGrowthPath(path: string) {
    setGrowthSelected((current) => {
      const next = new Set(current);
      if (next.has(path)) {
        next.delete(path);
      } else {
        next.add(path);
      }
      return next;
    });
  }

  function toggleDiff(path: string) {
    setExpandedDiffs((current) => {
      const next = new Set(current);
      if (next.has(path)) {
        next.delete(path);
      } else {
        next.add(path);
      }
      return next;
    });
    void loadChildDiffs(path);
  }

  async function loadChildDiffs(path: string) {
    if (!api || !compareResult || childDiffs[path] || inFlightChildPaths.current.has(path)) {
      return;
    }
    inFlightChildPaths.current.add(path);
    setLoadingChildPaths((current) => new Set(current).add(path));
    try {
      const result = await api.CompareSnapshotPath(compareResult.oldSnapshotId, compareResult.newSnapshotId, path);
      setChildDiffs((current) => ({ ...current, [path]: result.diffs ?? [] }));
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      inFlightChildPaths.current.delete(path);
      setLoadingChildPaths((current) => {
        const next = new Set(current);
        next.delete(path);
        return next;
      });
    }
  }

  function renderChildDiffRows(parentPath: string, level = 0): React.ReactNode {
    const rows = (childDiffs[parentPath] ?? []).filter(isChildDetailGrowthDiff);
    if (rows.length === 0) {
      return null;
    }
    return rows.map((child) => (
      <React.Fragment key={child.path}>
        <div className="childDiffRow" style={{ paddingLeft: 8 + level * 18 }} onClick={() => toggleDiff(child.path)} role="button" tabIndex={0}>
          <span className="childName">
            {expandedDiffs.has(child.path) ? <ChevronDown size={13} /> : <ChevronRight size={13} />}
            <Folder size={14} />
            <span title={child.path}>{fileName(child.path)}</span>
          </span>
          <span className={child.deltaBytes >= 0 ? 'growth' : 'shrink'}>
            {child.deltaBytes >= 0 ? '+' : ''}{formatSignedBytes(child.deltaBytes)}
          </span>
          <span>{formatBytes(child.newSize)}</span>
          <span className="diffActions">
            {child.cleanable && child.deltaBytes > 0 && (
              <input
                type="checkbox"
                checked={growthSelected.has(child.path)}
                onClick={(event) => event.stopPropagation()}
                onChange={() => toggleGrowthPath(child.path)}
                title="加入清理"
              />
            )}
            <button className="iconButton ghost" onClick={(event) => { event.stopPropagation(); openExplorer(child.path); }} type="button" title="在资源管理器中打开">
              <ExternalLink size={14} />
            </button>
          </span>
        </div>
        {expandedDiffs.has(child.path) && (
          <>
            {loadingChildPaths.has(child.path) && <span className="childLoading" style={{ paddingLeft: 8 + (level + 1) * 18 }}>正在加载下一层...</span>}
            {renderChildDiffRows(child.path, level + 1)}
            {childDiffs[child.path] && childDiffs[child.path].filter(isChildDetailGrowthDiff).length === 0 && (
              <span className="childLoading" style={{ paddingLeft: 8 + (level + 1) * 18 }}>没有更细一级的变化记录</span>
            )}
          </>
        )}
      </React.Fragment>
    ));
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
          <button className={`navItem ${activeView === 'cache' ? 'active' : ''}`} onClick={() => setActiveView('cache')} type="button" title="缓存清理">
            <Home size={19} />
            <span>缓存清理</span>
          </button>
          <button className={`navItem ${activeView === 'growth' ? 'active' : ''}`} onClick={() => setActiveView('growth')} type="button" title="目录增长分析">
            <BarChart3 size={19} />
            <span>增长分析</span>
          </button>
        </nav>

        {activeView === 'cache' ? (
          <section className="navSection" aria-label="清理分类">
            <p>Cleanup Categories</p>
            <div className="categoryNav">
              {categoryOrder.map((category) => {
                const Icon = categoryMeta[category].icon;
                const items = grouped.get(category) ?? [];
                const stats = categoryStats.get(category);
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
                    <small>{items.length > 0 ? formatBytes(stats?.sizeBytes ?? 0) : '-'}</small>
                    <em>{stats?.selectedCount ?? 0}/{items.length}</em>
                  </button>
                );
              })}
            </div>
          </section>
        ) : (
          <section className="navSection" aria-label="增长分析概览">
            <p>Growth Analysis</p>
            <div className="growthNavSummary">
              <FolderTree size={22} />
              <strong>{snapshots.length > 0 ? `${snapshots.length} 个快照` : '未建立快照'}</strong>
              <span>{compareResult ? '已选择两次快照对比' : '拍摄快照后选择两次记录'}</span>
              <small>{compareResult ? `${growthEntries.length} 个增长目录` : 'C 盘目录结构与大小'}</small>
            </div>
          </section>
        )}

        <div className="sideTools">
          <button className="plainNav" type="button" title="Settings">
            <Settings size={18} />
            <span>Settings</span>
          </button>
          <button className="plainNav" onClick={() => setAboutOpen(true)} type="button" title="About">
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

      {activeView === 'cache' ? (
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
              <button
                className="iconButton"
                type="button"
                onClick={allCategoriesExpanded ? collapseAllCategories : expandAllCategories}
                disabled={!scanResult || scanResult.items.length === 0}
                title={allCategoriesExpanded ? '收起全部目录' : '展开全部目录'}
              >
                {allCategoriesExpanded ? <FoldVertical size={18} /> : <UnfoldVertical size={18} />}
              </button>
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
              const stats = categoryStats.get(category);
              const allSelected = stats?.allSelected ?? false;
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
                    <strong>{formatBytes(stats?.sizeBytes ?? 0)}</strong>
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
      ) : (
        <section className="snapshotWorkspace">
          <header className="appHeader">
            <div>
              <p className="eyebrow">Disk Snapshot</p>
              <h2>C 盘空间变化分析</h2>
            </div>
            <div className="headerActions">
              <button className="button outline" onClick={loadSnapshots} disabled={busy !== null || !canUseBackend} type="button" title="刷新快照列表">
                {busy === 'list' ? <Loader2 className="spin" size={18} /> : <RefreshCw size={18} />}
                <span>刷新列表</span>
              </button>
              <button className="button primary" onClick={startGrowthScan} disabled={busy !== null || !canUseBackend} type="button" title="拍摄新快照">
                {busy === 'snapshot' ? <Loader2 className="spin" size={18} /> : <Camera size={18} />}
                <span>拍摄快照</span>
              </button>
            </div>
          </header>

          <div className="snapLabelRow">
            <input
              className="snapLabelInput"
              type="text"
              placeholder="快照标签，可选"
              value={labelInput}
              onChange={(event) => setLabelInput(event.target.value)}
              onKeyDown={(event) => {
                if (event.key === 'Enter') {
                  startGrowthScan();
                }
              }}
            />
          </div>

          {error && (
            <section className="notice error">
              <AlertTriangle size={18} />
              <span>{error}</span>
            </section>
          )}

          {growthCleanResult && (
            <section className="notice success">
              <CheckCircle2 size={18} />
              <span>
                已清理 {formatBytes(growthCleanResult.deletedBytes)}，删除 {growthCleanResult.deletedCount} 项，跳过 {growthCleanResult.skippedCount} 项
              </span>
            </section>
          )}

          <div className="snapMain">
            <section className="snapListPanel">
              <h3>快照历史 ({snapshots.length})</h3>
              {snapshots.length === 0 && (
                <div className="snapEmpty">
                  <Camera size={32} />
                  <span>暂无快照，先拍摄一次 C 盘空间状态。</span>
                </div>
              )}
              <div className="snapCards">
                {snapshots.map((snapshot) => (
                  <div
                    key={snapshot.id}
                    className={`snapCard ${newSnapshotID === snapshot.id ? 'selected' : ''}`}
                    onClick={() => setNewSnapshotID(snapshot.id)}
                    role="button"
                    tabIndex={0}
                    onKeyDown={(event) => {
                      if (event.key === 'Enter' || event.key === ' ') {
                        setNewSnapshotID(snapshot.id);
                      }
                    }}
                  >
                    <div className="snapCardLeft">
                      <Camera size={18} />
                      <div>
                        <strong>{snapshot.label}</strong>
                        <span>{formatBytes(snapshot.totalBytes)} / {snapshot.entryCount} 个目录</span>
                      </div>
                    </div>
                    <div className="snapCardRight">
                      <small>{formatTime(snapshot.createdAt)}</small>
                      <button
                        className="iconButton ghost"
                        onClick={(event) => {
                          event.stopPropagation();
                          deleteSnapshot(snapshot.id);
                        }}
                        type="button"
                        title="删除快照"
                      >
                        <Trash2 size={15} />
                      </button>
                    </div>
                  </div>
                ))}
              </div>
            </section>

            <section className="snapComparePanel">
              <h3>对比快照</h3>
              <div className="compareSelects">
                <div className="compareField">
                  <label>旧快照</label>
                  <select value={oldSnapshotID} onChange={(event) => setOldSnapshotID(event.target.value)}>
                    <option value="">选择基线快照</option>
                    {snapshots.map((snapshot) => (
                      <option key={snapshot.id} value={snapshot.id}>
                        {snapshot.label} ({formatTime(snapshot.createdAt)})
                      </option>
                    ))}
                  </select>
                </div>
                <div className="compareField">
                  <label>新快照</label>
                  <select value={newSnapshotID} onChange={(event) => setNewSnapshotID(event.target.value)}>
                    <option value="">选择当前快照</option>
                    {snapshots.map((snapshot) => (
                      <option key={snapshot.id} value={snapshot.id}>
                        {snapshot.label} ({formatTime(snapshot.createdAt)})
                      </option>
                    ))}
                  </select>
                </div>
              </div>
              <button className="button primary fullWidth" onClick={compareSnapshots} disabled={!oldSnapshotID || !newSnapshotID || oldSnapshotID === newSnapshotID || busy !== null} type="button" title="开始对比">
                {busy === 'compare' ? <Loader2 className="spin" size={18} /> : <TrendingUp size={18} />}
                <span>开始对比</span>
              </button>
            </section>
          </div>

          {compareResult && (
            <section className="compareResultPanel">
              <div className="compareSummary">
                <div className="compareStat">
                  <HardDrive size={20} />
                  <div>
                    <small>总空间变化</small>
                    <strong className={compareResult.newTotalBytes - compareResult.oldTotalBytes >= 0 ? 'growth' : 'shrink'}>
                      {compareResult.newTotalBytes - compareResult.oldTotalBytes >= 0 ? '+' : ''}
                      {formatSignedBytes(compareResult.newTotalBytes - compareResult.oldTotalBytes)}
                    </strong>
                  </div>
                </div>
                <div className="compareStat">
                  <TrendingUp size={20} />
                  <div>
                    <small>增长总量</small>
                    <strong className="growth">+{formatBytes(totalComparedGrowth)}</strong>
                    <small>{growthEntries.length} 个目录</small>
                  </div>
                </div>
                <div className="compareStat">
                  <TrendingDown size={20} />
                  <div>
                    <small>缩减总量</small>
                    <strong className="shrink">-{formatBytes(totalComparedShrink)}</strong>
                    <small>{shrinkEntries.length} 个目录</small>
                  </div>
                </div>
              </div>

              <h3>
                目录变化详情
                <small>{compareResult.oldLabel} {'>'} {compareResult.newLabel}</small>
              </h3>

              <div className="diffTable">
                <div className="diffHeader">
                  <span>目录</span>
                  <span>旧大小</span>
                  <span>新大小</span>
                  <span>变化量</span>
                  <span>变化率</span>
                  <span>操作</span>
                </div>
                {snapshotDiffs.map((diff) => {
                  const isExpanded = expandedDiffs.has(diff.path);
                  const dirName = fileName(diff.path);
                  return (
                    <React.Fragment key={diff.path}>
                      <div className={`diffRow ${diff.deltaBytes > 0 ? 'growth' : diff.deltaBytes < 0 ? 'shrink' : ''}`} onClick={() => toggleDiff(diff.path)} role="button" tabIndex={0}>
                        <span className="diffName">
                          {isExpanded ? <ChevronDown size={14} /> : <ChevronRight size={14} />}
                          <Folder size={16} />
                          <span>{dirName}</span>
                        </span>
                        <span className="diffSize">{formatBytes(diff.oldSize)}</span>
                        <span className="diffSize">{formatBytes(diff.newSize)}</span>
                        <span className={`diffDelta ${diff.deltaBytes >= 0 ? 'growth' : 'shrink'}`}>
                          {diff.deltaBytes >= 0 ? '+' : ''}{formatSignedBytes(diff.deltaBytes)}
                        </span>
                        <span className={`diffPct ${diff.deltaPercent >= 0 ? 'growth' : 'shrink'}`}>
                          {diff.deltaPercent >= 0 ? '+' : ''}{diff.deltaPercent}%
                        </span>
                        <span className="diffActions">
                          {diff.cleanable && diff.deltaBytes > 0 && (
                            <input
                              type="checkbox"
                              checked={growthSelected.has(diff.path)}
                              onClick={(event) => event.stopPropagation()}
                              onChange={() => toggleGrowthPath(diff.path)}
                              title="加入清理"
                            />
                          )}
                          <button className="iconButton ghost" onClick={(event) => { event.stopPropagation(); openExplorer(diff.path); }} type="button" title="在资源管理器中打开">
                            <ExternalLink size={15} />
                          </button>
                        </span>
                      </div>
                      {isExpanded && (
                        <div className="diffDetail">
                          <div className="diffDetailContent">
                            <p><strong>完整路径：</strong>{diff.path}</p>
                            <p><strong>旧大小：</strong>{formatBytes(diff.oldSize)}</p>
                            <p><strong>新大小：</strong>{formatBytes(diff.newSize)}</p>
                            <p><strong>变化：</strong><span className={diff.deltaBytes >= 0 ? 'growth' : 'shrink'}>{diff.deltaBytes >= 0 ? '+' : ''}{formatSignedBytes(diff.deltaBytes)} ({diff.deltaPercent >= 0 ? '+' : ''}{diff.deltaPercent}%)</span></p>
                            <div className="childDiffs">
                              {loadingChildPaths.has(diff.path) && <span className="childLoading">正在加载下一层...</span>}
                              {renderChildDiffRows(diff.path)}
                              {childDiffs[diff.path] && childDiffs[diff.path].filter(isChildDetailGrowthDiff).length === 0 && (
                                <span className="childLoading">没有更细一级的变化记录</span>
                              )}
                            </div>
                          </div>
                        </div>
                      )}
                    </React.Fragment>
                  );
                })}
              </div>
            </section>
          )}

          <footer className="bottomBar">
            <button className="button secondary" type="button" title="分析选项">
              <SlidersHorizontal size={18} />
              <span>分析选项</span>
            </button>
            <div className="scanStatus">
              <CheckCircle2 size={17} />
              <span>{compareResult ? '对比完成' : snapshots.length > 0 ? '已有快照' : '未扫描'}</span>
              {lastGrowthScanAt && <em>{lastGrowthScanAt.toLocaleString('zh-CN', { hour: '2-digit', minute: '2-digit' })}</em>}
            </div>
            <div className="bottomActions">
              <button className="button outline" onClick={loadSnapshots} disabled={busy !== null || !canUseBackend} type="button" title="刷新列表">
                <RefreshCw size={18} />
                <span>刷新</span>
              </button>
              <button className="button danger" onClick={() => setGrowthConfirmOpen(true)} disabled={busy !== null || selectedGrowthEntries.length === 0} type="button" title="清理可安全删除的增长目录">
                {busy === 'growthClean' ? <Loader2 className="spin" size={18} /> : <Trash2 size={18} />}
                <span>清理所选 ({formatBytes(selectedGrowthBytes)})</span>
              </button>
            </div>
          </footer>
        </section>
      )}

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

      {aboutOpen && (
        <div className="modalBackdrop" role="presentation">
          <section className="confirmDialog aboutDialog" role="dialog" aria-modal="true" aria-label="About">
            <button className="closeButton" onClick={() => setAboutOpen(false)} type="button" title="关闭">
              <X size={18} />
            </button>
            <div className="confirmIcon aboutIcon">
              <Info size={28} />
            </div>
            <h2>About</h2>
            <div className="aboutDetails">
              <span>{aboutTime}</span>
              <strong>By BussanQ</strong>
            </div>
            <div className="dialogActions">
              <button className="button primary" onClick={() => setAboutOpen(false)} type="button">
                OK
              </button>
            </div>
          </section>
        </div>
      )}

      {growthConfirmOpen && (
        <div className="modalBackdrop" role="presentation">
          <section className="confirmDialog" role="dialog" aria-modal="true" aria-label="确认清理增长目录">
            <button className="closeButton" onClick={() => setGrowthConfirmOpen(false)} type="button" title="关闭">
              <X size={18} />
            </button>
            <div className="confirmIcon">
              <Trash2 size={28} />
            </div>
            <h2>确认清理所选增长目录？</h2>
            <p>将清理 {selectedGrowthEntries.length} 个被标记为缓存或临时数据的目录，共 {formatBytes(selectedGrowthBytes)}。系统目录和普通用户资料目录不会被允许删除。</p>
            <div className="dialogActions">
              <button className="button secondary" onClick={() => setGrowthConfirmOpen(false)} type="button">
                取消
              </button>
              <button className="button danger" onClick={confirmGrowthClean} type="button">
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

function normalizeScanResult(result: ScanResult): ScanResult {
  return {
    ...result,
    items: result.items ?? [],
    summaries: result.summaries ?? [],
    failures: result.failures ?? [],
  };
}

function normalizeCleanResult(result: CleanResult): CleanResult {
  return {
    ...result,
    failures: result.failures ?? [],
  };
}

function normalizeSnapshotCompareResult(result: SnapshotCompareResult): SnapshotCompareResult {
  return {
    ...result,
    diffs: result.diffs ?? [],
  };
}

function formatSignedBytes(bytes: number) {
  if (!Number.isFinite(bytes)) {
    return '0 B';
  }
  if (bytes < 0) {
    return `-${formatBytes(Math.abs(bytes))}`;
  }
  return formatBytes(bytes);
}

function formatTime(iso: string) {
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) {
    return iso;
  }
  return date.toLocaleString('zh-CN', { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' });
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
