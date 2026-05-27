import { formatBps, formatDuration } from "@/lib/format";



type Props = {

  trafficDown: number;

  trafficUp: number;

  sessionSec: number;

  visible: boolean;

};



export function StatRow({ trafficDown, trafficUp, sessionSec, visible }: Props) {

  if (!visible) return null;

  return (

    <div className="stat-panel grid grid-cols-2 gap-3 text-center">

      <div>

        <p className="text-caption text-muted">↓</p>

        <p className="font-mono text-stat text-fg">{formatBps(trafficDown)}</p>

      </div>

      <div>

        <p className="text-caption text-muted">↑</p>

        <p className="font-mono text-stat text-fg">{formatBps(trafficUp)}</p>

      </div>

      <div className="col-span-2 flex items-center justify-center gap-1.5 text-caption text-muted">

        <span aria-hidden>⏱</span>

        <span className="font-mono text-body text-fg">

          {formatDuration(sessionSec)}

        </span>

      </div>

    </div>

  );

}

