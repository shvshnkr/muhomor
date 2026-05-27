import { useEffect, useState } from "react";

import { Button } from "@/components/ui/button";

import { ConnectPowerButton } from "@/components/ConnectPowerButton";

import { StatusPill } from "@/components/StatusPill";

import { ServerCard } from "@/components/ServerCard";

import { StatRow } from "@/components/StatRow";

import { Toast } from "@/components/Toast";

import { Collapsible } from "@/components/Collapsible";

import { useSessionTimer } from "@/hooks/useSessionTimer";

import { useConnectionStore } from "@/stores/connection";

import {

  ClearPinnedError,

  ConnectAction,

  ExportLog,

  Ping,

} from "../wailsjs/go/main/App";



type Props = {

  onExtended?: () => void;

  embedded?: boolean;

};



export function Simple({ onExtended, embedded }: Props) {

  const { conn, settings, toast: serverToast, clearToast } = useConnectionStore();

  const [pingBusy, setPingBusy] = useState(false);

  const [localPing, setLocalPing] = useState<string | null>(null);

  const [localPingErr, setLocalPingErr] = useState(false);

  const [infoToast, setInfoToast] = useState("");



  const keepErrors = !!settings?.uiKeepErrorsOnScreen;

  const liveError = conn?.errorText?.trim() ?? "";



  const sessionSec = useSessionTimer(!!conn?.connected);



  useEffect(() => {

    setLocalPing(null);

    setLocalPingErr(false);

  }, [conn?.state, conn?.connected, conn?.profileId, conn?.proxyName]);



  useEffect(() => {

    if (!liveError && !keepErrors) {

      clearToast();

    }

  }, [liveError, keepErrors, clearToast]);



  if (!conn) {

    return (

      <div className="flex h-full items-center justify-center text-muted">

        Загрузка…

      </div>

    );

  }



  const pingText = localPing ?? conn.pingResult;

  const pingErr = localPingErr || conn.pingError;



  const pillVariant = conn.connected

    ? "connected"

    : liveError

      ? "error"

      : conn.connecting

        ? "connecting"

        : "idle";



  const btnState = conn.connectDanger

    ? "danger"

    : conn.connecting

      ? "connecting"

      : conn.connected

        ? "connected"

        : "idle";



  const showActivity =

    (conn.connecting || liveError !== "") && !!conn.activityText;



  async function onConnect() {

    await ConnectAction();

  }



  async function onPing() {

    setPingBusy(true);

    try {

      const r = await Ping();

      setLocalPing(r.text);

      setLocalPingErr(r.error);

    } finally {

      setPingBusy(false);

    }

  }



  async function onExport() {

    try {

      const path = await ExportLog();

      setInfoToast(`Лог: ${path}`);

    } catch (e) {

      setInfoToast(String(e));

    }

  }



  const serverInfo =

    serverToast?.level !== "error" ? serverToast?.text?.trim() ?? "" : "";

  const toastText = liveError || infoToast || serverInfo;

  const toastIsError = !!liveError;



  function onDismissToast() {

    clearToast();

    setInfoToast("");

    if (liveError) {

      void ClearPinnedError();

    }

  }



  return (

    <div className="flex h-full min-h-0 flex-col gap-4 p-4">

      <div className="hero-surface flex flex-col gap-4 rounded-2xl p-4">

        <StatusPill

          label={conn.statusTitle}

          variant={pillVariant}

          activity={showActivity ? conn.activityText : undefined}

        />



        <ServerCard

          profileName={conn.profileName}

          connected={conn.connected}

          pingText={pingBusy ? "Проверка…" : pingText}

          pingError={pingErr}

          muted={!conn.connected}

          pingEnabled={conn.pingEnabled}

          pingBusy={pingBusy}

          onPing={onPing}

        />



        <ConnectPowerButton

          label={conn.connectLabel}

          state={btnState}

          disabled={conn.busy && conn.connected && !conn.connecting}

          onClick={onConnect}

        />



        <StatRow

          visible={conn.connected}

          trafficDown={conn.trafficDown ?? 0}

          trafficUp={conn.trafficUp ?? 0}

          sessionSec={sessionSec}

        />

      </div>



      {conn.probeText ? (

        <Collapsible title="Диагностика">

          <pre className="max-h-40 overflow-y-auto whitespace-pre-wrap text-caption text-muted">

            {conn.probeText}

          </pre>

        </Collapsible>

      ) : null}



      <Toast

        text={toastText}

        error={toastIsError}

        onDismiss={toastText ? onDismissToast : undefined}

      />



      <div className="mt-auto flex items-center justify-between gap-2 border-t border-white/[0.06] pt-3">

        <Button

          variant="secondary"

          className="!min-h-9 !w-auto text-sm"

          onClick={onExport}

        >

          Экспорт лога

        </Button>

        {!embedded && onExtended ? (

          <button

            type="button"

            className="focus-ring text-sm text-accent hover:text-accent-hover"

            onClick={onExtended}

          >

            Расширенный режим →

          </button>

        ) : null}

      </div>

    </div>

  );

}

