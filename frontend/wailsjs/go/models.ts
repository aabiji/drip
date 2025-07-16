export namespace p2p {
	
	export class FileChunk {
	    transfer_id: string;
	    data: number[];
	    chunkIndex: number;
	
	    static createFrom(source: any = {}) {
	        return new FileChunk(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.transfer_id = source["transfer_id"];
	        this.data = source["data"];
	        this.chunkIndex = source["chunkIndex"];
	    }
	}
	export class TransferInfo {
	    transfer_id: string;
	    recipients: string[];
	    name: string;
	    size: number;
	    numChunks: number;
	    chunkSize: number;
	
	    static createFrom(source: any = {}) {
	        return new TransferInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.transfer_id = source["transfer_id"];
	        this.recipients = source["recipients"];
	        this.name = source["name"];
	        this.size = source["size"];
	        this.numChunks = source["numChunks"];
	        this.chunkSize = source["chunkSize"];
	    }
	}

}

