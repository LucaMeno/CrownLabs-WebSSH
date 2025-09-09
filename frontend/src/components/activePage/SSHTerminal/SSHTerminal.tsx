import React, { useEffect, useContext, useRef } from 'react';
import { useParams } from 'react-router-dom';
import { useXTerm } from 'react-xtermjs';
import { AuthContext } from '../../../contexts/AuthContext';
import './SSHTerminal.css';
import { FitAddon } from 'xterm-addon-fit';

const SSHTerminal: React.FC = () => {
  const { namespace = '', VMname: nomeVM = '', environment = '' } = useParams();
  const { ref, instance } = useXTerm();
  const { token } = useContext(AuthContext);

  const fitRef = useRef<FitAddon | null>(null);
  const resizeObs = useRef<ResizeObserver | null>(null);
  const wsRef = useRef<WebSocket | null>(null);

  useEffect(() => {
    if (!instance) return;

    instance.options = {
      cursorBlink: true,
      scrollback: 10000,
      convertEol: true,
      theme: { background: '#000000' },
    };

    if (!fitRef.current) {
      fitRef.current = new FitAddon();
      instance.loadAddon(fitRef.current);
    }


    const fitAndNotify = () =>
      requestAnimationFrame(() => {
        fitRef.current?.fit();
        const { cols, rows } = instance;
        if (wsRef.current?.readyState === WebSocket.OPEN) {
          wsRef.current.send(JSON.stringify({ type: 'resize', cols, rows }));
        }
      });

    instance.focus();
    fitAndNotify();
    window.addEventListener('resize', fitAndNotify);
    (document as any).fonts?.ready?.then?.(fitAndNotify);

    if (ref.current && 'ResizeObserver' in window) {
      resizeObs.current = new ResizeObserver(fitAndNotify);
      resizeObs.current.observe(ref.current);
    }

    // --- WebSocket ---
    const IP = 'localhost';
    const PORT = 8090;
    const socketUrl = `ws://${IP}:${PORT}/webssh`;
    const ws = new WebSocket(socketUrl);
    wsRef.current = ws;

    ws.onopen = () => {

      console.log("ENV:", environment);

      // init message
      ws.send(JSON.stringify({
        namespace,
        vmName: nomeVM,
        token,
        InitialWidth: instance.cols,
        InitialHeight: instance.rows,
        Environment: environment
      }));

      instance.writeln('');
      instance.writeln('\x1b[1;36m📡 Connecting to VM... \x1b[0m');
      instance.writeln('\x1b[1;32m[✔] SSH connection success.\x1b[0m\r\n');

      fitAndNotify();

      setInterval(() => ws.send(JSON.stringify({ type: 'ping' })), 10000);
    };

    ws.onmessage = (ev) => {
      const obj = JSON.parse(ev.data);
      if (obj.error) {
        instance.write(`\r\n\x1b[1;31m${obj.error}\x1b[0m\r\n`);
        ws.close();
        return;
      }
      if (obj.data) instance.write(obj.data);
    };

    ws.onerror = () => {
      instance.writeln('\x1b[1;31m[✖] Connection error.\x1b[0m\r\n');
    };
    ws.onclose = () => {
      instance.writeln('\x1b[1;33m[●] Connection closed.\x1b[0m\r\n');
    };


    const disposeData = instance.onData((data) => {
      ws.readyState === WebSocket.OPEN &&
        ws.send(JSON.stringify({ type: 'input', data }));
    });


    return () => {
      disposeData.dispose();
      resizeObs.current?.disconnect();
      window.removeEventListener('resize', fitAndNotify);
      try { ws.close(); } catch { }
      wsRef.current = null;
      try { instance.dispose(); } catch { }
    };
  }, [instance, namespace, nomeVM, token, ref]);

  return <div ref={ref} className="ssh-terminal" />;
};

export default SSHTerminal;
