export namespace agent {
	
	export class Segment {
	    start: number;
	    end: number;
	    text: string;
	    confidence: number;
	
	    static createFrom(source: any = {}) {
	        return new Segment(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.start = source["start"];
	        this.end = source["end"];
	        this.text = source["text"];
	        this.confidence = source["confidence"];
	    }
	}
	export class MeetingState {
	    audio_file_path: string;
	    language: string;
	    wav_file_path: string;
	    raw_transcript: string;
	    segments: Segment[];
	    audio_duration: number;
	    meeting_notes: string;
	    summary_content?: string;
	    summary_error?: string;
	    output_path: string;
	    transcript_path?: string;
	    html_output_path?: string;
	    markdown_path?: string;
	    summary_enabled: boolean;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new MeetingState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.audio_file_path = source["audio_file_path"];
	        this.language = source["language"];
	        this.wav_file_path = source["wav_file_path"];
	        this.raw_transcript = source["raw_transcript"];
	        this.segments = this.convertValues(source["segments"], Segment);
	        this.audio_duration = source["audio_duration"];
	        this.meeting_notes = source["meeting_notes"];
	        this.summary_content = source["summary_content"];
	        this.summary_error = source["summary_error"];
	        this.output_path = source["output_path"];
	        this.transcript_path = source["transcript_path"];
	        this.html_output_path = source["html_output_path"];
	        this.markdown_path = source["markdown_path"];
	        this.summary_enabled = source["summary_enabled"];
	        this.error = source["error"];
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

export namespace main {
	
	export class AppInfo {
	    engine: string;
	    whisperModel: string;
	    llmModel: string;
	
	    static createFrom(source: any = {}) {
	        return new AppInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.engine = source["engine"];
	        this.whisperModel = source["whisperModel"];
	        this.llmModel = source["llmModel"];
	    }
	}
	export class DeviceInfo {
	    id: string;
	    name: string;
	    type: string;
	
	    static createFrom(source: any = {}) {
	        return new DeviceInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.type = source["type"];
	    }
	}

}

