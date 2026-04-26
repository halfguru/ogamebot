import type { JSX } from 'solid-js';

interface HeaderProps {
  connected: boolean;
  lastUpdate: Date | null;
}

export default function Header(props: HeaderProps) {
  return (
    <header>
      <div class="header-left">
        <h1>⭐ OGame Bot</h1>
      </div>
      <div class="header-right">
        <span class={`status ${props.connected ? 'connected' : 'disconnected'}`}>
          {props.connected ? '● Connected' : '○ Disconnected'}
        </span>
        {props.lastUpdate && (
          <span class="last-update">
            Updated: {props.lastUpdate.toLocaleTimeString()}
          </span>
        )}
      </div>
    </header>
  );
}
