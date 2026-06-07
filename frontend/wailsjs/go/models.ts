export namespace model {
	
	export class CategorySummary {
	    category: string;
	    itemCount: number;
	    sizeBytes: number;
	
	    static createFrom(source: any = {}) {
	        return new CategorySummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.category = source["category"];
	        this.itemCount = source["itemCount"];
	        this.sizeBytes = source["sizeBytes"];
	    }
	}
	export class CleanFailure {
	    path: string;
	    reason: string;
	
	    static createFrom(source: any = {}) {
	        return new CleanFailure(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.reason = source["reason"];
	    }
	}
	export class CleanRequest {
	    taskId: string;
	    itemIds: string[];
	
	    static createFrom(source: any = {}) {
	        return new CleanRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.taskId = source["taskId"];
	        this.itemIds = source["itemIds"];
	    }
	}
	export class CleanResult {
	    deletedCount: number;
	    deletedBytes: number;
	    skippedCount: number;
	    failures: CleanFailure[];
	    cancelled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new CleanResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.deletedCount = source["deletedCount"];
	        this.deletedBytes = source["deletedBytes"];
	        this.skippedCount = source["skippedCount"];
	        this.failures = this.convertValues(source["failures"], CleanFailure);
	        this.cancelled = source["cancelled"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class DirEntry {
	    path: string;
	    sizeBytes: number;
	    fileCount: number;
	
	    static createFrom(source: any = {}) {
	        return new DirEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.sizeBytes = source["sizeBytes"];
	        this.fileCount = source["fileCount"];
	    }
	}
	export class DiskGrowthEntry {
	    path: string;
	    name: string;
	    sizeBytes: number;
	    previousSizeBytes: number;
	    growthBytes: number;
	    growthPercent: number;
	    depth: number;
	    fileCount: number;
	    dirCount: number;
	    trend: string;
	    cleanable: boolean;
	    defaultSelected: boolean;
	
	    static createFrom(source: any = {}) {
	        return new DiskGrowthEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.name = source["name"];
	        this.sizeBytes = source["sizeBytes"];
	        this.previousSizeBytes = source["previousSizeBytes"];
	        this.growthBytes = source["growthBytes"];
	        this.growthPercent = source["growthPercent"];
	        this.depth = source["depth"];
	        this.fileCount = source["fileCount"];
	        this.dirCount = source["dirCount"];
	        this.trend = source["trend"];
	        this.cleanable = source["cleanable"];
	        this.defaultSelected = source["defaultSelected"];
	    }
	}
	export class DiskGrowthOptions {
	    root: string;
	    maxDepth: number;
	    maxResults: number;
	    minGrowthBytes: number;
	
	    static createFrom(source: any = {}) {
	        return new DiskGrowthOptions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.root = source["root"];
	        this.maxDepth = source["maxDepth"];
	        this.maxResults = source["maxResults"];
	        this.minGrowthBytes = source["minGrowthBytes"];
	    }
	}
	export class ScanFailure {
	    path: string;
	    reason: string;
	
	    static createFrom(source: any = {}) {
	        return new ScanFailure(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.reason = source["reason"];
	    }
	}
	export class DiskGrowthResult {
	    taskId: string;
	    snapshotId: string;
	    root: string;
	    scannedAt: string;
	    previousSnapshotId: string;
	    previousScannedAt: string;
	    hasBaseline: boolean;
	    totalBytes: number;
	    totalGrowthBytes: number;
	    dirCount: number;
	    fileCount: number;
	    entries: DiskGrowthEntry[];
	    failures: ScanFailure[];
	    cancelled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new DiskGrowthResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.taskId = source["taskId"];
	        this.snapshotId = source["snapshotId"];
	        this.root = source["root"];
	        this.scannedAt = source["scannedAt"];
	        this.previousSnapshotId = source["previousSnapshotId"];
	        this.previousScannedAt = source["previousScannedAt"];
	        this.hasBaseline = source["hasBaseline"];
	        this.totalBytes = source["totalBytes"];
	        this.totalGrowthBytes = source["totalGrowthBytes"];
	        this.dirCount = source["dirCount"];
	        this.fileCount = source["fileCount"];
	        this.entries = this.convertValues(source["entries"], DiskGrowthEntry);
	        this.failures = this.convertValues(source["failures"], ScanFailure);
	        this.cancelled = source["cancelled"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class DiskSnapshot {
	    id: string;
	    createdAt: string;
	    label: string;
	    drive: string;
	    entries: DirEntry[];
	    totalBytes: number;
	
	    static createFrom(source: any = {}) {
	        return new DiskSnapshot(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.createdAt = source["createdAt"];
	        this.label = source["label"];
	        this.drive = source["drive"];
	        this.entries = this.convertValues(source["entries"], DirEntry);
	        this.totalBytes = source["totalBytes"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class GrowthCleanRequest {
	    snapshotId: string;
	    paths: string[];
	
	    static createFrom(source: any = {}) {
	        return new GrowthCleanRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.snapshotId = source["snapshotId"];
	        this.paths = source["paths"];
	    }
	}
	
	export class ScanItem {
	    id: string;
	    path: string;
	    sizeBytes: number;
	    modifiedAt: string;
	    category: string;
	    risk: string;
	    defaultSelected: boolean;
	    isDirectory: boolean;
	    isVirtual: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ScanItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.path = source["path"];
	        this.sizeBytes = source["sizeBytes"];
	        this.modifiedAt = source["modifiedAt"];
	        this.category = source["category"];
	        this.risk = source["risk"];
	        this.defaultSelected = source["defaultSelected"];
	        this.isDirectory = source["isDirectory"];
	        this.isVirtual = source["isVirtual"];
	    }
	}
	export class ScanOptions {
	    categories: string[];
	
	    static createFrom(source: any = {}) {
	        return new ScanOptions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.categories = source["categories"];
	    }
	}
	export class ScanResult {
	    taskId: string;
	    items: ScanItem[];
	    summaries: CategorySummary[];
	    totalCount: number;
	    totalBytes: number;
	    failures: ScanFailure[];
	    cancelled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ScanResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.taskId = source["taskId"];
	        this.items = this.convertValues(source["items"], ScanItem);
	        this.summaries = this.convertValues(source["summaries"], CategorySummary);
	        this.totalCount = source["totalCount"];
	        this.totalBytes = source["totalBytes"];
	        this.failures = this.convertValues(source["failures"], ScanFailure);
	        this.cancelled = source["cancelled"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SnapshotDiff {
	    path: string;
	    oldSize: number;
	    newSize: number;
	    deltaBytes: number;
	    deltaPercent: number;
	    cleanable: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SnapshotDiff(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.oldSize = source["oldSize"];
	        this.newSize = source["newSize"];
	        this.deltaBytes = source["deltaBytes"];
	        this.deltaPercent = source["deltaPercent"];
	        this.cleanable = source["cleanable"];
	    }
	}
	export class SnapshotCompareResult {
	    oldSnapshotId: string;
	    newSnapshotId: string;
	    oldLabel: string;
	    newLabel: string;
	    oldTotalBytes: number;
	    newTotalBytes: number;
	    diffs: SnapshotDiff[];
	
	    static createFrom(source: any = {}) {
	        return new SnapshotCompareResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.oldSnapshotId = source["oldSnapshotId"];
	        this.newSnapshotId = source["newSnapshotId"];
	        this.oldLabel = source["oldLabel"];
	        this.newLabel = source["newLabel"];
	        this.oldTotalBytes = source["oldTotalBytes"];
	        this.newTotalBytes = source["newTotalBytes"];
	        this.diffs = this.convertValues(source["diffs"], SnapshotDiff);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class SnapshotInfo {
	    id: string;
	    createdAt: string;
	    label: string;
	    totalBytes: number;
	    entryCount: number;
	
	    static createFrom(source: any = {}) {
	        return new SnapshotInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.createdAt = source["createdAt"];
	        this.label = source["label"];
	        this.totalBytes = source["totalBytes"];
	        this.entryCount = source["entryCount"];
	    }
	}
	export class SnapshotPathCompareResult {
	    oldSnapshotId: string;
	    newSnapshotId: string;
	    path: string;
	    diffs: SnapshotDiff[];
	
	    static createFrom(source: any = {}) {
	        return new SnapshotPathCompareResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.oldSnapshotId = source["oldSnapshotId"];
	        this.newSnapshotId = source["newSnapshotId"];
	        this.path = source["path"];
	        this.diffs = this.convertValues(source["diffs"], SnapshotDiff);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

