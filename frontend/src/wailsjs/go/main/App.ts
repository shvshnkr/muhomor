import { Call } from "../../runtime/runtime";

export interface ConnectionDTO {
  state: string;
  connected: boolean;
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
}

export interface SettingsDTO {
  serviceMode: string;
  mixedPort: number;
  routeQuick: number;
  multipathEnabled: boolean;
  multipathPreset: string;
  multipathWlEmergencyOnly: boolean;
  wlBuiltinConnectEnabled: boolean;
  uiKeepErrorsOnScreen: boolean;
}

export interface PingResultDTO {
  text: string;
  error: boolean;
}

export interface GroupDTO {
  id: number;
  name: string;
  kind: string;
  subscriptionLink: string;
  label: string;
}

export interface ProfileDTO {
  id: number;
  name: string;
  type: string;
  groupId: number;
  enabled: boolean;
  lastDelayMs: number;
}

function invoke<T>(method: string, ...args: unknown[]): Promise<T> {
  return Call(method, ...args) as Promise<T>;
}

export function GetConnectionSnapshot(): Promise<ConnectionDTO> {
  return invoke("GetConnectionSnapshot");
}

export function GetSettingsSnapshot(): Promise<SettingsDTO> {
  return invoke("GetSettingsSnapshot");
}

export function ConnectAction(): Promise<string> {
  return invoke("ConnectAction");
}

export function ClearPinnedError(): Promise<void> {
  return invoke("ClearPinnedError");
}

export function Ping(): Promise<PingResultDTO> {
  return invoke("Ping");
}

export function ExportLog(): Promise<string> {
  return invoke("ExportLog");
}

export function ShowWindow(): Promise<void> {
  return invoke("ShowWindow");
}

export function Quit(): Promise<void> {
  return invoke("Quit");
}

export function ListGroups(): Promise<GroupDTO[]> {
  return invoke("ListGroups");
}

export function ListProfiles(): Promise<ProfileDTO[]> {
  return invoke("ListProfiles");
}

export function GetRouteQuick(): Promise<number> {
  return invoke("GetRouteQuick");
}

export function SetRouteQuick(v: number): Promise<void> {
  return invoke("SetRouteQuick", v);
}

export function LoadSettingsJSON(): Promise<Record<string, unknown>> {
  return invoke("LoadSettingsJSON");
}

export function SaveSettingsJSON(patch: Record<string, unknown>): Promise<void> {
  return invoke("SaveSettingsJSON", patch);
}

export function ReloadMihomo(): Promise<void> {
  return invoke("ReloadMihomo");
}

export function BulkPingAll(): Promise<string> {
  return invoke("BulkPingAll");
}

export function RefreshGroup(groupID: number): Promise<string> {
  return invoke("RefreshGroup", groupID);
}

export function UpdateGroupSubscription(groupID: number, link: string): Promise<void> {
  return invoke("UpdateGroupSubscription", groupID, link);
}

export function AddGroupServer(groupID: number, uri: string): Promise<void> {
  return invoke("AddGroupServer", groupID, uri);
}

export function StopDaemon(): Promise<void> {
  return invoke("StopDaemon");
}

export function StartDaemon(): Promise<void> {
  return invoke("StartDaemon");
}

export function SocketPath(): Promise<string> {
  return invoke("SocketPath");
}

export function GroupKinds(): Promise<string[]> {
  return invoke("GroupKinds");
}
