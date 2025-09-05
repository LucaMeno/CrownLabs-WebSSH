import React, { useEffect, useContext } from 'react';
import { useParams } from 'react-router-dom';
import { useXTerm } from 'react-xtermjs';
import { AuthContext } from '../../../contexts/AuthContext';
import './SSHTerminal.css';

const SSHTerminal: React.FC = () => {
  const { namespace = '', VMname: nomeVM = '' } = useParams();
  const { ref, instance } = useXTerm();
  const { token } = useContext(AuthContext);

  useEffect(() => {
    if (!instance) return;

    instance.options = {
      cursorBlink: true,
      scrollback: 10000,
      theme: {
        background: '#000000',
      },
    };

    instance.focus();
    const IP = 'localhost'
    const PORT = 8090;
    const socketUrl = `ws://${IP}:${PORT}/webssh`;
    const ws = new WebSocket(socketUrl);
    //const ws = new WebSocket(`wss://950.staging.crownlabs.polito.it/webssh`);

    ws.onopen = () => {
      ws.send(
        JSON.stringify({
          namespace,
          vmName: nomeVM,
          token,
          InitialWidth: instance.cols,
          InitialHeight: instance.rows
        })
      );

      instance.writeln(`\x1b[1;36m📡 Connecting to VM \x1b[0m`);
      instance.writeln('[✔] SSH connection success.\r\n');

      setInterval(() => {
        const obj = { type: "ping" };
        console.log(obj);
        ws.send(JSON.stringify(obj));
      }, 10000);
    };

    ws.onmessage = (ev) => {
      var obj = JSON.parse(ev.data);

      console.log("Received message:", obj);

      if (obj.error) {
        instance.write(`\r\n\x1b[1;31m${obj.error}\x1b[0m\r\n`);
        ws.close();
        return;
      }
      if (obj.data) {
        instance.write(obj.data);
      }
    };

    ws.onerror = () => {
      instance.writeln('[✖] Connection error.\r\n');
    };

    ws.onclose = () => {
      instance.writeln('[●] Connection closed.\r\n');
    };

    instance.onData((data) => {
      const msg = {
        type: "input",
        data: data,
      };

      console.log("Terminal data:", msg);
      ws.send(JSON.stringify(msg));
    });

    instance.onResize(({ cols, rows }) => {
      const msg = {
        type: "resize",
        cols: cols,
        rows: rows,
      };
      console.log("Terminal resized:", msg);
      ws.send(JSON.stringify(msg));
    });

    return () => {
      ws.close();
      instance.dispose();
    };
  }, [instance, namespace, nomeVM, token]);

  return <div ref={ref} className="ssh-terminal" />;
};

export default SSHTerminal;
