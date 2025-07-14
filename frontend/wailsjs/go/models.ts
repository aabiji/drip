export namespace p2p {
	
	export class FileChunk {
	    data: number[];
	    chunkIndex: number;
	    recipients: string[];
	
	    static createFrom(source: any = {}) {
	        return new FileChunk(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.data = source["data"];
	        this.chunkIndex = source["chunkIndex"];
	        this.recipients = source["recipients"];
	    }
	}
	export class FileInfo {
	    recipients: string[];
	    name: string;
	    size: number;
	    numChunks: number;
	
	    static createFrom(source: any = {}) {
	        return new FileInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.recipients = source["recipients"];
	        this.name = source["name"];
	        this.size = source["size"];
	        this.numChunks = source["numChunks"];
	    }
	}

}

