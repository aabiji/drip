export namespace p2p {
	
	export class FileChunk {
	    transferId: string;
	    recipient: string;
	    data: number[];
	    offset: number;
	
	    static createFrom(source: any = {}) {
	        return new FileChunk(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.transferId = source["transferId"];
	        this.recipient = source["recipient"];
	        this.data = source["data"];
	        this.offset = source["offset"];
	    }
	}
	export class TransferInfo {
	    transferId: string;
	    recipient: string;
	    name: string;
	    size: number;
	
	    static createFrom(source: any = {}) {
	        return new TransferInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.transferId = source["transferId"];
	        this.recipient = source["recipient"];
	        this.name = source["name"];
	        this.size = source["size"];
	    }
	}

}

