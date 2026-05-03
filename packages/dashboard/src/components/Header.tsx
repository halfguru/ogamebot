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
        <span class="header-server">OGameX Automation Engine</span>
      </div>
      <div class="header-right">
        <div class="bot-indicators">
          <Show when={props.connected}>
            <span class="bot-indicator">
              <span class="indicator-dot" />
              Defender
            </span>
            <span class="bot-indicator">
              <span class="indicator-dot" />
              Builder
            </span>
            <span class="bot-indicator">
              <span class="indicator-dot" />
              Farmer
            </span>
          </Show>
        </div>
        <span class={`status ${props.connected ? 'connected' : 'disconnected'}`}>
          <span class="status-dot" />
          {props.connected ? 'Connected' : 'Disconnected'}
        </span>
        {props.lastUpdate && (
          <span class="last-update">
            {props.lastUpdate.toLocaleTimeString()}
          </span>
        )}
      </div>
    </header>
  );
}
