export namespace wailsapp {
	
	export class ConnectionDTO {
	    state: string;
	    connected: boolean;
	    connectedVerified: boolean;
	    connectedDegraded: boolean;
	    verificationPhase: string;
	    profileId: number;
	    profileName: string;
	    proxyName: string;
	    statusTitle: string;
	    activityText: string;
	    probeText: string;
	    busy: boolean;
	    connecting: boolean;
	    errorText: string;
	    connectLabel: string;
	    connectDanger: boolean;
	    pingLabel: string;
	    pingResult: string;
	    pingError: boolean;
	    pingEnabled: boolean;
	    trafficUp: number;
	    trafficDown: number;
	
	    static createFrom(source: any = {}) {
	        return new ConnectionDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.state = source["state"];
	        this.connected = source["connected"];
	        this.connectedVerified = source["connectedVerified"];
	        this.connectedDegraded = source["connectedDegraded"];
	        this.verificationPhase = source["verificationPhase"];
	        this.profileId = source["profileId"];
	        this.profileName = source["profileName"];
	        this.proxyName = source["proxyName"];
	        this.statusTitle = source["statusTitle"];
	        this.activityText = source["activityText"];
	        this.probeText = source["probeText"];
	        this.busy = source["busy"];
	        this.connecting = source["connecting"];
	        this.errorText = source["errorText"];
	        this.connectLabel = source["connectLabel"];
	        this.connectDanger = source["connectDanger"];
	        this.pingLabel = source["pingLabel"];
	        this.pingResult = source["pingResult"];
	        this.pingError = source["pingError"];
	        this.pingEnabled = source["pingEnabled"];
	        this.trafficUp = source["trafficUp"];
	        this.trafficDown = source["trafficDown"];
	    }
	}
	export class GroupDTO {
	    id: number;
	    name: string;
	    kind: string;
	    subscriptionLink: string;
	    label: string;
	
	    static createFrom(source: any = {}) {
	        return new GroupDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.kind = source["kind"];
	        this.subscriptionLink = source["subscriptionLink"];
	        this.label = source["label"];
	    }
	}
	export class PingResultDTO {
	    text: string;
	    error: boolean;
	
	    static createFrom(source: any = {}) {
	        return new PingResultDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.text = source["text"];
	        this.error = source["error"];
	    }
	}
	export class ProfileDTO {
	    id: number;
	    name: string;
	    type: string;
	    groupId: number;
	    enabled: boolean;
	    lastDelayMs: number;
	
	    static createFrom(source: any = {}) {
	        return new ProfileDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.type = source["type"];
	        this.groupId = source["groupId"];
	        this.enabled = source["enabled"];
	        this.lastDelayMs = source["lastDelayMs"];
	    }
	}
	export class SettingsDTO {
	    serviceMode: string;
	    mixedPort: number;
	    routeQuick: number;
	    multipathEnabled: boolean;
	    multipathPreset: string;
	    multipathWlEmergencyOnly: boolean;
	    wlBuiltinConnectEnabled: boolean;
	    uiKeepErrorsOnScreen: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SettingsDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.serviceMode = source["serviceMode"];
	        this.mixedPort = source["mixedPort"];
	        this.routeQuick = source["routeQuick"];
	        this.multipathEnabled = source["multipathEnabled"];
	        this.multipathPreset = source["multipathPreset"];
	        this.multipathWlEmergencyOnly = source["multipathWlEmergencyOnly"];
	        this.wlBuiltinConnectEnabled = source["wlBuiltinConnectEnabled"];
	        this.uiKeepErrorsOnScreen = source["uiKeepErrorsOnScreen"];
	    }
	}

}

