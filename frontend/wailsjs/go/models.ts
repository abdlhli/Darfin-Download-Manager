export namespace models {
	
	export class Segment {
	    index: number;
	    startByte: number;
	    endByte: number;
	    downloadedBytes: number;
	    tempFilePath: string;
	    completed: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Segment(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.index = source["index"];
	        this.startByte = source["startByte"];
	        this.endByte = source["endByte"];
	        this.downloadedBytes = source["downloadedBytes"];
	        this.tempFilePath = source["tempFilePath"];
	        this.completed = source["completed"];
	    }
	}
	export class DownloadItem {
	    id: string;
	    url: string;
	    fileName: string;
	    savePath: string;
	    totalSize: number;
	    downloadedSize: number;
	    cookies: string;
	    referrer: string;
	    status: string;
	    segments: Segment[];
	    threadCount: number;
	    speed: number;
	    progress: number;
	    resumable: boolean;
	    error?: string;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    updatedAt: any;
	    // Go type: time
	    completedAt?: any;
	
	    static createFrom(source: any = {}) {
	        return new DownloadItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.url = source["url"];
	        this.fileName = source["fileName"];
	        this.savePath = source["savePath"];
	        this.totalSize = source["totalSize"];
	        this.downloadedSize = source["downloadedSize"];
	        this.cookies = source["cookies"];
	        this.referrer = source["referrer"];
	        this.status = source["status"];
	        this.segments = this.convertValues(source["segments"], Segment);
	        this.threadCount = source["threadCount"];
	        this.speed = source["speed"];
	        this.progress = source["progress"];
	        this.resumable = source["resumable"];
	        this.error = source["error"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
	        this.completedAt = this.convertValues(source["completedAt"], null);
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
	
	export class Settings {
	    maxConcurrentDownloads: number;
	    defaultThreadCount: number;
	    defaultSaveDir: string;
	    speedLimitEnabled: boolean;
	    speedLimitBytesPerSec: number;
	    autoStartDownload: boolean;
	    smartCategorization: boolean;
	    autoExtract: boolean;
	    bandwidthMode: string;
	    prioritySecondaryLimit: number;
	
	    static createFrom(source: any = {}) {
	        return new Settings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.maxConcurrentDownloads = source["maxConcurrentDownloads"];
	        this.defaultThreadCount = source["defaultThreadCount"];
	        this.defaultSaveDir = source["defaultSaveDir"];
	        this.speedLimitEnabled = source["speedLimitEnabled"];
	        this.speedLimitBytesPerSec = source["speedLimitBytesPerSec"];
	        this.autoStartDownload = source["autoStartDownload"];
	        this.smartCategorization = source["smartCategorization"];
	        this.autoExtract = source["autoExtract"];
	        this.bandwidthMode = source["bandwidthMode"];
	        this.prioritySecondaryLimit = source["prioritySecondaryLimit"];
	    }
	}

}

