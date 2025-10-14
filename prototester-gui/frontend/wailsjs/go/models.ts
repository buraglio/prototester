export namespace main {
	
	export class TestRequest {
	    protocol: string;
	    target4: string;
	    target6: string;
	    hostname: string;
	    port: number;
	    count: number;
	    interval: number;
	    timeout: number;
	    size: number;
	    dnsProtocol: string;
	    dnsQuery: string;
	    ipv4Only: boolean;
	    ipv6Only: boolean;
	    verbose: boolean;
	
	    static createFrom(source: any = {}) {
	        return new TestRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.protocol = source["protocol"];
	        this.target4 = source["target4"];
	        this.target6 = source["target6"];
	        this.hostname = source["hostname"];
	        this.port = source["port"];
	        this.count = source["count"];
	        this.interval = source["interval"];
	        this.timeout = source["timeout"];
	        this.size = source["size"];
	        this.dnsProtocol = source["dnsProtocol"];
	        this.dnsQuery = source["dnsQuery"];
	        this.ipv4Only = source["ipv4Only"];
	        this.ipv6Only = source["ipv6Only"];
	        this.verbose = source["verbose"];
	    }
	}
	export class HistoryEntry {
	    id: string;
	    name: string;
	    // Go type: time
	    timestamp: any;
	    request: TestRequest;
	    result?: tester.TestResult;
	
	    static createFrom(source: any = {}) {
	        return new HistoryEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.timestamp = this.convertValues(source["timestamp"], null);
	        this.request = this.convertValues(source["request"], TestRequest);
	        this.result = this.convertValues(source["result"], tester.TestResult);
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
	export class SavedConfig {
	    id: string;
	    name: string;
	    // Go type: time
	    createdAt: any;
	    config: TestRequest;
	
	    static createFrom(source: any = {}) {
	        return new SavedConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.config = this.convertValues(source["config"], TestRequest);
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

export namespace tester {
	
	export class Statistics {
	    sent: number;
	    received: number;
	    lost: number;
	    min_ms: number;
	    max_ms: number;
	    avg_ms: number;
	    stddev_ms: number;
	    jitter_ms: number;
	    success_rate: number;
	
	    static createFrom(source: any = {}) {
	        return new Statistics(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sent = source["sent"];
	        this.received = source["received"];
	        this.lost = source["lost"];
	        this.min_ms = source["min_ms"];
	        this.max_ms = source["max_ms"];
	        this.avg_ms = source["avg_ms"];
	        this.stddev_ms = source["stddev_ms"];
	        this.jitter_ms = source["jitter_ms"];
	        this.success_rate = source["success_rate"];
	    }
	}
	export class ComparisonResult {
	    tcp_v4_stats?: Statistics;
	    tcp_v6_stats?: Statistics;
	    udp_v4_stats?: Statistics;
	    udp_v6_stats?: Statistics;
	    dns_v4_stats?: Statistics;
	    dns_v6_stats?: Statistics;
	    http_v4_stats?: Statistics;
	    http_v6_stats?: Statistics;
	    icmp_v4_stats?: Statistics;
	    icmp_v6_stats?: Statistics;
	    ipv4_score: number;
	    ipv6_score: number;
	    winner: string;
	    resolved_ipv4: string;
	    resolved_ipv6: string;
	    protocol: string;
	    hostname: string;
	    port: number;
	    dns_query?: string;
	    // Go type: time
	    timestamp: any;
	
	    static createFrom(source: any = {}) {
	        return new ComparisonResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.tcp_v4_stats = this.convertValues(source["tcp_v4_stats"], Statistics);
	        this.tcp_v6_stats = this.convertValues(source["tcp_v6_stats"], Statistics);
	        this.udp_v4_stats = this.convertValues(source["udp_v4_stats"], Statistics);
	        this.udp_v6_stats = this.convertValues(source["udp_v6_stats"], Statistics);
	        this.dns_v4_stats = this.convertValues(source["dns_v4_stats"], Statistics);
	        this.dns_v6_stats = this.convertValues(source["dns_v6_stats"], Statistics);
	        this.http_v4_stats = this.convertValues(source["http_v4_stats"], Statistics);
	        this.http_v6_stats = this.convertValues(source["http_v6_stats"], Statistics);
	        this.icmp_v4_stats = this.convertValues(source["icmp_v4_stats"], Statistics);
	        this.icmp_v6_stats = this.convertValues(source["icmp_v6_stats"], Statistics);
	        this.ipv4_score = source["ipv4_score"];
	        this.ipv6_score = source["ipv6_score"];
	        this.winner = source["winner"];
	        this.resolved_ipv4 = source["resolved_ipv4"];
	        this.resolved_ipv6 = source["resolved_ipv6"];
	        this.protocol = source["protocol"];
	        this.hostname = source["hostname"];
	        this.port = source["port"];
	        this.dns_query = source["dns_query"];
	        this.timestamp = this.convertValues(source["timestamp"], null);
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
	
	export class TestConfig {
	    target_ipv4?: string;
	    target_ipv6?: string;
	    hostname?: string;
	    port: number;
	    count: number;
	    interval: number;
	    timeout: number;
	    size?: number;
	    dns_protocol?: string;
	    dns_query?: string;
	    ipv4_only: boolean;
	    ipv6_only: boolean;
	    verbose: boolean;
	
	    static createFrom(source: any = {}) {
	        return new TestConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.target_ipv4 = source["target_ipv4"];
	        this.target_ipv6 = source["target_ipv6"];
	        this.hostname = source["hostname"];
	        this.port = source["port"];
	        this.count = source["count"];
	        this.interval = source["interval"];
	        this.timeout = source["timeout"];
	        this.size = source["size"];
	        this.dns_protocol = source["dns_protocol"];
	        this.dns_query = source["dns_query"];
	        this.ipv4_only = source["ipv4_only"];
	        this.ipv6_only = source["ipv6_only"];
	        this.verbose = source["verbose"];
	    }
	}
	export class TestResult {
	    mode: string;
	    protocol: string;
	    targets: Record<string, string>;
	    ipv4_results?: Statistics;
	    ipv6_results?: Statistics;
	    comparison?: ComparisonResult;
	    test_config: TestConfig;
	    // Go type: time
	    timestamp: any;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new TestResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mode = source["mode"];
	        this.protocol = source["protocol"];
	        this.targets = source["targets"];
	        this.ipv4_results = this.convertValues(source["ipv4_results"], Statistics);
	        this.ipv6_results = this.convertValues(source["ipv6_results"], Statistics);
	        this.comparison = this.convertValues(source["comparison"], ComparisonResult);
	        this.test_config = this.convertValues(source["test_config"], TestConfig);
	        this.timestamp = this.convertValues(source["timestamp"], null);
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

