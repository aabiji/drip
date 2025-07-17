export namespace p2p {
	
	export class FileChunk {
	    transferId: string;
	    recipients: string[];
	    data: number[];
	
	    static createFrom(source: any = {}) {
	        return new FileChunk(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.transferId = source["transferId"];
	        this.recipients = source["recipients"];
	        this.data = source["data"];
	    }
	}
	export class TransferInfo {
	    transferId: string;
	    recipients: string[];
	    name: string;
	    size: number;
	
	    static createFrom(source: any = {}) {
	        return new TransferInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.transferId = source["transferId"];
	        this.recipients = source["recipients"];
	        this.name = source["name"];
	        this.size = source["size"];
	    }
	}

}

