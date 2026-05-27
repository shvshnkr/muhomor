import { useEffect, useState } from "react";

import { Simple } from "./pages/Simple";

import { Extended } from "./pages/Extended";

import { ShutdownOverlay } from "@/components/ShutdownOverlay";
import { WindowChrome } from "@/components/WindowChrome";
import { useShutdown } from "@/stores/shutdown";

import { WindowCenter, WindowSetSize } from "./wailsjs/runtime/runtime";



const windowSimple = { w: 420, h: 612 };

const windowExtended = { w: 1000, h: 752 };



export default function App() {
  const { shutdown, shuttingDown } = useShutdown();
  const [extended, setExtended] = useState(false);



  useEffect(() => {

    const size = extended ? windowExtended : windowSimple;

    WindowSetSize(size.w, size.h);

    WindowCenter();

  }, [extended]);



  return (
    <div className="relative flex h-full min-h-0 flex-col">
      {!shuttingDown && <WindowChrome title="muhomor" />}
      <div className="min-h-0 flex-1 overflow-hidden">
        {!shuttingDown &&
          (extended ? (
            <Extended onBack={() => setExtended(false)} />
          ) : (
            <Simple onExtended={() => setExtended(true)} />
          ))}
      </div>
      <ShutdownOverlay state={shutdown} />
    </div>

  );

}

