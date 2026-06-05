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

}

