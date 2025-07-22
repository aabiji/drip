export namespace p2p {
	
	export class SessionCancel {
	    sessionId: string;
	    recipients: string[];
	
	    static createFrom(source: any = {}) {
	        return new SessionCancel(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sessionId = source["sessionId"];
	        this.recipients = source["recipients"];
	    }
	}
	export class Transfer {
	    transferId: string;
	    recipient: string;
	    size: number;
	
	    static createFrom(source: any = {}) {
	        return new Transfer(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.transferId = source["transferId"];
	        this.recipient = source["recipient"];
	        this.size = source["size"];
	    }
	}
	export class SessionInfo {
	    sessionId: string;
	    recipients: string[];
	    transfers: Transfer[];
	    sender?: string;
	
	    static createFrom(source: any = {}) {
	        return new SessionInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sessionId = source["sessionId"];
	        this.recipients = source["recipients"];
	        this.transfers = this.convertValues(source["transfers"], Transfer);
	        this.sender = source["sender"];
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
	
	export class TransferChunk {
	    transferId: string;
	    recipient: string;
	    data: number[];
	    offset: number;
	
	    static createFrom(source: any = {}) {
	        return new TransferChunk(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.transferId = source["transferId"];
	        this.recipient = source["recipient"];
	        this.data = source["data"];
	        this.offset = source["offset"];
	    }
	}

}

